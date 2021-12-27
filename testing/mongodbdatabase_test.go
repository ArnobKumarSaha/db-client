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
		By("Clean MongoDBDatabase and its dependants first")
		err := to.DeleteMongoDBDatabaseSchema()
		Expect(err).NotTo(HaveOccurred())
		to.WaitForMongoDBDBDDatabaseAndDependantsCleanup().Should(Succeed())
		By("Cleaning up every other resources those were created for testing purpose")
		err = to.CleanUpEverything()
		Expect(err).NotTo(HaveOccurred())
		to.CheckIfEverythingIsCleaned().Should(Succeed())
	})

	var runner = func() {
		By("Creating VaultServer")
		err := to.CreateVaultServer()
		Expect(err).NotTo(HaveOccurred())

		By("Creating MongoDB")
		err = to.CreateMongoDB()
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for both MongoDB & vaultServer to be ready")
		to.CheckReadiness().Should(Succeed())

		By("Running Job(operator)")
		err = to.CreateRunnerJob()
		Expect(err).NotTo(HaveOccurred())

		By("Creating schema")
		err = to.CreateMongoDBDatabaseSchema()
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for success of MongoDBDatabase schema")
		to.CheckSuccessOfSchema().Should(Succeed())
	}

	var set = func() {
		to.Mongodb = to.GetMongoDBSpec(to.DBOptions)
		to.SchemaDatabase = to.GetSchemaMongoDBDatabaseSpec(to.SchemaOptions)
	}

	var approvalTrue = func() {
		BeforeEach(func() {
			to.AutoApproval = true
			set()
		})
		It("should run successfully", runner)
	}
	var approvalFalse = func() {
		BeforeEach(func() {
			to.AutoApproval = false
			set()
		})
		It("should run successfully", runner)
	}

	var initAndApprovalCase = func() {
		BeforeEach(func() {
			to.ToRestore = false
		})
		Context("autoApproval on", approvalTrue)
		Context("autoApproval off", approvalFalse)
	}
	var restoreAndApprovalCase = func() {
		BeforeEach(func() {
			to.ToRestore = true
		})
		Context("autoApproval on", approvalTrue)
		Context("autoApproval off", approvalFalse)
	}

	// namespace * dbType * init-restore * autoApproval
	Context("In the same namespace... ", func() {
		// StandAlone
		Context("With StandAlone Database", func() {
			BeforeEach(func() {
				to.DBType = framework.StandAlone
			})
			Context("For Init", initAndApprovalCase)
			Context("For Restore", restoreAndApprovalCase)
		})

		// ReplicaSet
		FContext("With Replicaset", func() {
			BeforeEach(func() {
				to.DBType = framework.ReplicaSet
			})
			Context("For Init", initAndApprovalCase)
			Context("For Restore", restoreAndApprovalCase)
		})

		// Sharding
		Context("With Sharding", func() {
			BeforeEach(func() {
				to.DBType = framework.Sharded
			})
			Context("For Init", initAndApprovalCase)
			Context("For Restore", restoreAndApprovalCase)
		})
	})
})
