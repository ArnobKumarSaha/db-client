package schema_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"kubedb.dev/schema-manager/controllers/schema/framework"
)

var _ = Describe("Mongodbdatabase", func() {
	var (
		to framework.TestOptions
		f  *framework.Invocation
	)
	BeforeEach(func() {
		f = framework.NewInvocation()
		to = framework.TestOptions{
			Invocation:     f,
			Mongodb:        f.GetMongoDBSpec(),
			Vault:          f.GetVaultServerSpec(),
			SchemaDatabase: f.GetSchemaMongoDBDatabaseSpec(),
			InitJob:        f.GetTheRunnerJob(),
			RestoreSession: nil,
			DBOptions: &framework.DBOptions{
				DBType:         framework.StandAlone,
				SslModeEnabled: false,
			},
			SchemaOptions: &framework.SchemaOptions{
				AutoApproval: true,
				ToRestore:    false,
			},
			Secret: nil,
		}
	})
	AfterEach(func() {
		By("Cleaning up every resources those were created for testing purpose")
		to.CleanUpEverything().Should(Succeed())
		to.CheckIfEverythingIsCleaned().Should(Succeed())
	})

	var runner = func() {
		By("Creating VaultServer")
		err := to.CreateVaultServer()
		Expect(err).NotTo(HaveOccurred())

		By("Creating MongoDB")
		err = to.CreateMongoDB()
		Expect(err).NotTo(HaveOccurred())

		By("Creating schema")
		err = to.CreateMongoDBDatabaseSchema()
		Expect(err).NotTo(HaveOccurred())

		to.CheckReadiness().Should(Succeed())

		By("Readiness checked; Running Job")
		err = to.CreateRunnerJob()
		Expect(err).NotTo(HaveOccurred())
		to.CheckCompletenessOfInit().Should(Succeed())

		By("Waiting for success of MongoDBDatabase schema")
		to.CheckSuccessOfSchema().Should(Succeed())
	}

	var set = func() {
		to.Mongodb = to.GetMongoDBSpec(to.DBOptions)
		to.SchemaDatabase = to.GetSchemaMongoDBDatabaseSpec(to.SchemaOptions)
	}

	Context("In the same namespace... ", func() {
		// StandAlone
		Context("With StandAlone Database", func() {
			It("For Initialization", runner)
			Context("For Restore", func() {
				BeforeEach(func() {
					to.ToRestore = true
					set()
				})
				It("should run successfully", runner)
			})
		})

		// ReplicaSet
		Context("With Replicaset", func() {
			BeforeEach(func() {
				to.DBType = framework.ReplicaSet
			})
			FContext("For Init", func() {
				BeforeEach(func() {
					to.ToRestore = false
					set()
				})
				It("should run successfully", runner)
			})
			Context("For Restore", func() {
				BeforeEach(func() {
					to.ToRestore = true
					set()
				})
				It("should run successfully", runner)
			})
		})

		// Sharding
		Context("With Sharding", func() {
			BeforeEach(func() {
				to.DBType = framework.Sharded
			})
			Context("For Init", func() {
				BeforeEach(func() {
					to.ToRestore = false
					set()
				})
				It("should run successfully", runner)
			})
			Context("For Restore", func() {
				BeforeEach(func() {
					to.ToRestore = true
					set()
				})
				It("should run successfully", runner)
			})
		})
	})
})

/*
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
*/
