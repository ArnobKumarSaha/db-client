/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Free Trial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Free-Trial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package schema

import (
	"context"
	"flag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	appcat "kmodules.xyz/custom-resources/client/clientset/versioned/scheme"
	schemav1alpha1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	"kubedb.dev/schema-manager/controllers/schema/framework"
	kubevaultscheme "kubevault.dev/apimachinery/client/clientset/versioned/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	stashScheme "stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clientGoScheme "k8s.io/client-go/kubernetes/scheme"
	"kmodules.xyz/client-go/tools/clientcmd"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	scheme = runtime.NewScheme()
)

const (
	TIMEOUT = 20 * time.Minute
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(TIMEOUT)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

func init() {
	utilruntime.Must(clientGoScheme.AddToScheme(scheme))
	utilruntime.Must(schemav1alpha1.AddToScheme(scheme))
	utilruntime.Must(kubedbscheme.AddToScheme(scheme))
	utilruntime.Must(kubevaultscheme.AddToScheme(scheme))
	utilruntime.Must(stashScheme.AddToScheme(scheme))
	utilruntime.Must(appcat.AddToScheme(scheme))
}

func getManager() manager.Manager {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	tm := time.Minute * 3
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "82ac5e1d.kubedb.com",
		SyncPeriod:             &tm,
	})
	Expect(err).NotTo(HaveOccurred())
	return mgr
}

var _ = BeforeSuite(func() {
	config, err := clientcmd.BuildConfigFromContext("", "")
	Expect(err).NotTo(HaveOccurred())
	mgr := getManager()
	go func() {
		err = mgr.Start(context.TODO())
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	// Clients
	kubeClient := kubernetes.NewForConfigOrDie(config)
	myClient := mgr.GetClient()
	framework.RootFramework = framework.New(config, kubeClient, myClient)

	//+kubebuilder:scaffold:scheme

	// Create namespace
	By("Using namespace " + framework.RootFramework.Namespace())
	err = framework.RootFramework.CreateNamespace()
	Expect(err).NotTo(HaveOccurred())

	framework.RootFramework.EventuallyCRD().Should(Succeed())

}, 60)

var _ = AfterSuite(func() {
	By("Deleting Namespace")
	err := framework.RootFramework.DeleteNamespace()
	Expect(err).NotTo(HaveOccurred())
})
