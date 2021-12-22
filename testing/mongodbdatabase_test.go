package schema_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	smv1a1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	"kubedb.dev/schema-manager/controllers/schema/framework"
	kvm_server "kubevault.dev/apimachinery/apis/kubevault/v1alpha1"
)

var _ = Describe("Mongodbdatabase", func() {
	var (
		f           *framework.Invocation
		vault       *kvm_server.VaultServer
		mongo       *kdm.MongoDB
		schemaMongo *smv1a1.MongoDBDatabase
		job         *batchv1.Job
	)
	BeforeEach(func() {
		f = framework.NewInvocation().Invoke()
		vault = f.GetVaultServerSpec()
		mongo = f.GetMongoShardSpec()
		schemaMongo = f.GetSchemaMongoDBDatabaseSpec()
		job = f.GetTheRunnerJob()
	})
	AfterEach(func() {
		By("Deleting MongoDBDatabase schema")
		err := f.DeleteMongoDBDatabaseSchema(schemaMongo)
		Expect(err).NotTo(HaveOccurred())

		By("Deleting Runner Job")
		err = f.DeleteRunnerJob(job)
		Expect(err).NotTo(HaveOccurred())

		By("Deleting Vault & MongoDB")
		err = f.DeleteVaultServer(vault)
		Expect(err).NotTo(HaveOccurred())
		err = f.DeleteMongoDB(mongo)
		Expect(err).NotTo(HaveOccurred())
	})
	Context("In same namespace for Mongo shard", func() {
		It("Checking Init", func() {
			By("Creating VaultServer")
			err := f.CreateVaultServer(vault)
			Expect(err).NotTo(HaveOccurred())

			By("Creating MongoDB")
			err = f.CreateMongoDB(mongo)
			Expect(err).NotTo(HaveOccurred())

			By("Creating schema")
			err = f.CreateMongoDBDatabaseSchema(schemaMongo)
			Expect(err).NotTo(HaveOccurred())

			f.CheckReadiness().Should(Succeed())

			By("Creating Runner Job")
			err = f.CreateRunnerJob(job)
			Expect(err).NotTo(HaveOccurred())

			f.CheckCompletenessOfInit().Should(Succeed())
		})
	})
})
