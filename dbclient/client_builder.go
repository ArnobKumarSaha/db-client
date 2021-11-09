package dbclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"

	"go.mongodb.org/mongo-driver/mongo"
	mgoptions "go.mongodb.org/mongo-driver/mongo/options"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/tools/certholder"
)

type KubeDBClientBuilder struct {
	kubeClient kubernetes.Interface
	db         *api.MongoDB
	url        string
	podName    string
	repSetName string
	direct     bool
	certs      *certholder.ResourceCerts
	ctx        context.Context
}

func NewKubeDBClientBuilder(db *api.MongoDB, kubeClient kubernetes.Interface) *KubeDBClientBuilder {
	return &KubeDBClientBuilder{
		kubeClient: kubeClient,
		db:         db,
		direct:     false,
	}
}

func (o *KubeDBClientBuilder) WithURL(url string) *KubeDBClientBuilder {
	o.url = url
	return o
}

func (o *KubeDBClientBuilder) WithPod(podName string) *KubeDBClientBuilder {
	o.podName = podName
	o.direct = true
	return o
}

func (o *KubeDBClientBuilder) WithReplSet(replSetName string) *KubeDBClientBuilder {
	o.repSetName = replSetName
	return o
}

func (o *KubeDBClientBuilder) WithContext(ctx context.Context) *KubeDBClientBuilder {
	o.ctx = ctx
	return o
}

func (o *KubeDBClientBuilder) WithDirect() *KubeDBClientBuilder {
	o.direct = true
	return o
}

func (o *KubeDBClientBuilder) WithCerts(certs *certholder.ResourceCerts) *KubeDBClientBuilder {
	o.certs = certs
	return o
}

/*

 */
func (o *KubeDBClientBuilder) GetMongoClient() (*Client, error) {
	db := o.db

	if o.podName != "" {
		o.url = o.getURL()
	}

	fmt.Println("i am here")
	if o.podName == "" && o.url == "" {
		if db.Spec.ShardTopology != nil {
			// Shard
			o.url = strings.Join(db.MongosHosts(), ",")
		} else {
			// Standalone or ReplicaSet
			o.url = strings.Join(db.Hosts(), ",")
		}
	}

	clientOpts, err := o.getMongoDBClientOpts()
	if err != nil {
		return nil, err
	}
	fmt.Println("client opts ran successful")

	client, err := mongo.Connect(o.ctx, clientOpts)
	if err != nil {
		return nil, err
	}
	fmt.Println("MongoDB connected")

	// it was nil in place of readpref.Primary(), that also does not work
	err = client.Ping(o.ctx, nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println("Pinging successful")

	return &Client{
		Client: client,
	}, nil
}

func (o *KubeDBClientBuilder) getURL() string {
	nodeType := o.podName[:strings.LastIndex(o.podName, "-")]
	return fmt.Sprintf("%s.%s.%s.svc", o.podName, o.db.GoverningServiceName(nodeType), o.db.Namespace)
}

func (o *KubeDBClientBuilder) getMongoDBClientOpts() (*mgoptions.ClientOptions, error) {
	db := o.db
	repSetConfig := ""
	if o.repSetName != "" {
		repSetConfig = "replicaSet=" + o.repSetName + "&"
	}

	user, pass, err := o.getMongoDBRootCredentials()
	if err != nil {
		return nil, err
	}
	var clientOpts *mgoptions.ClientOptions
	if db.Spec.TLS != nil {
		secretName := db.GetCertSecretName(api.MongoDBClientCert, "")
		var paths *certholder.Paths
		if o.certs == nil {
			certSecret, err := o.kubeClient.CoreV1().Secrets(db.Namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
			if err != nil {
				klog.Error(err, "failed to get certificate secret. ", secretName)
				return nil, err
			}

			certs, _ := certholder.DefaultHolder.
				ForResource(api.SchemeGroupVersion.WithResource(api.ResourcePluralMongoDB), db.ObjectMeta)
			_, err = certs.Save(certSecret)
			if err != nil {
				klog.Error(err, "failed to save certificate")
				return nil, err
			}

			paths, err = certs.Get(secretName)
			if err != nil {
				return nil, err
			}
		} else {
			paths, err = o.certs.Get(secretName)
			if err != nil {
				return nil, err
			}
		}

		uri := fmt.Sprintf("mongodb://%s:%s@%s/admin?%vtls=true&tlsCAFile=%v&tlsCertificateKeyFile=%v", user, pass, o.url, repSetConfig, paths.CACert, paths.Pem)
		clientOpts = mgoptions.Client().ApplyURI(uri)
	} else {
		clientOpts = mgoptions.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s/admin?%v", user, pass, o.url, repSetConfig))
	}
	fmt.Println("clientOpts = ", clientOpts)
	clientOpts.SetDirect(o.direct)
	clientOpts.SetConnectTimeout(5 * time.Second)

	return clientOpts, nil
}

/*
Get the corresponding auth secret &
Returns username and password of mongoDb root admin
 */
func (o *KubeDBClientBuilder) getMongoDBRootCredentials() (string, string, error) {
	db := o.db
	if db.Spec.AuthSecret == nil {
		return "", "", errors.New("no database secret")
	}
	secret, err := o.kubeClient.CoreV1().Secrets(db.Namespace).Get(context.TODO(), db.Spec.AuthSecret.Name, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	fmt.Println("getMongoDBRootCredentials, no error")
	return string(secret.Data[core.BasicAuthUsernameKey]), string(secret.Data[core.BasicAuthPasswordKey]), nil
}
