package schema_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"kubedb.dev/schema-manager/controllers/schema/framework"
	"time"
)

var _ = XDescribe("Mongodb-database Testing", func() {
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
			RunnerJob:      f.GetTheRunnerJob(),
			DBOptions: &framework.DBOptions{
				DBType:         framework.StandAlone,
				SslModeEnabled: false,
			},
			SchemaOptions: &framework.SchemaOptions{
				AutoApproval:       true,
				ToRestore:          false,
				SchemaDatabaseName: framework.MongoDBDatabaseSchemaName,
				VolumeSourceSecret: false,
			},
			VaultOptions: &framework.VaultOptions{},
		}
	})
	AfterEach(func() {
		By("Clean MongoDBDatabase and its dependants first")
		err := to.DeleteMongoDBDatabaseSchema()
		Expect(err).NotTo(HaveOccurred())
		to.WaitForMongoDBDDatabaseAndDependantsCleanup().Should(Succeed())
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
	var setAll = func() {
		to.Vault = to.GetVaultServerSpec()
		set()
	}

	// approval separation
	var (
		approvalTrue = func() {
			BeforeEach(func() {
				to.AutoApproval = true
				setAll()
			})
			It("should run successfully", runner)
		}
		approvalFalse = func() {
			BeforeEach(func() {
				to.AutoApproval = false
				setAll()
			})
			XIt("should run successfully", runner)
		}
	)

	// Init-restore separation
	var (
		initAndApprovalCase = func() {
			BeforeEach(func() {
				to.ToRestore = false
			})
			Context("autoApproval on", approvalTrue)
			Context("autoApproval off", approvalFalse)
		}
		restoreAndApprovalCase = func() {
			BeforeEach(func() {
				to.ToRestore = true
			})
			Context("autoApproval on", approvalTrue)
			Context("autoApproval off", approvalFalse)
		}
	)

	// DBType separation
	var (
		standAloneCase = func() {
			BeforeEach(func() {
				to.DBType = framework.StandAlone
			})
			Context("For Init", initAndApprovalCase)
			XContext("For Restore", restoreAndApprovalCase)
		}
		replicasetCase = func() {
			BeforeEach(func() {
				to.DBType = framework.ReplicaSet
			})
			Context("For Init", initAndApprovalCase)
			XContext("For Restore", restoreAndApprovalCase)
		}
		shardCase = func() {
			BeforeEach(func() {
				to.DBType = framework.Sharded
			})
			Context("For Init", initAndApprovalCase)
			Context("For Restore", restoreAndApprovalCase)
		}
	)

	// namespace * tls * dbType * init-restore * autoApproval
	// cases calculated : 2   *  2  *   3  *  2  *  2
	// to pass in Github : 2  *  1  *   3  *  2  *  1
	XContext("In the same namespace... ", func() {
		BeforeEach(func() {
			f.SetSameNamespace()
			to.Invocation = f
		})
		Context("TLS enabled", func() {
			BeforeEach(func() {
				to.SslModeEnabled = true
			})
			Context("With StandAlone Database", standAloneCase)
			Context("With Replicaset", replicasetCase)
			Context("With Sharding", shardCase)
		})
		Context("TLS disabled", func() {
			BeforeEach(func() {
				to.SslModeEnabled = false
			})
			Context("With StandAlone Database", standAloneCase)
			Context("With Replicaset", replicasetCase)
			Context("With Sharding", shardCase)
		})
	})

	// In different namespace
	XContext("In different namespace... ", func() {
		XContext("TLS enabled", func() {
			BeforeEach(func() {
				to.SslModeEnabled = true
			})
			Context("With StandAlone Database", standAloneCase)
			Context("With Replicaset", replicasetCase)
			Context("With Sharding", shardCase)
		})
		Context("TLS disabled", func() {
			BeforeEach(func() {
				to.SslModeEnabled = false
			})
			Context("With StandAlone Database", standAloneCase)
			Context("With Replicaset", replicasetCase)
			XContext("With Sharding", shardCase)
		})
	})

	Context("database-name conflict Case", func() {
		BeforeEach(func() {
			to.SslModeEnabled = false
			to.DBType = framework.ReplicaSet
			to.SchemaDatabaseName = framework.MongoDBDatabaseSchemaName
			to.ToRestore = false
			to.AutoApproval = true
			set()
		})
		Context("Checking Conflict", func() {
			It("should succeeded", func() {
				runner()
				to.SchemaDatabaseName = framework.MongoDBDatabaseSchemaName + "-2"
				to.SchemaDatabase = to.GetSchemaMongoDBDatabaseSpec(to.SchemaOptions)

				By("Creating another schema")
				err := to.CreateMongoDBDatabaseSchema()
				Expect(err).NotTo(HaveOccurred())

				By("Waiting to be ignored of MongoDBDatabase schema")
				to.CheckIgnoredOfSchema().Should(Succeed())

				time.Sleep(30 * time.Second)
				err = to.DeleteMongoDBDatabaseSchemaByName(framework.MongoDBDatabaseSchemaName)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		FContext("try with secret", func() {
			BeforeEach(func() {
				to.VolumeSourceSecret = true
				set()
			})
			It("should be succeeded", runner)
		})
	})
})

var _ = Describe("Taking BackUp", func() {
	var (
		to framework.TestOptions
		f  *framework.Invocation
	)
	BeforeEach(func() {
		f = framework.NewInvocation()
		to = framework.TestOptions{
			Invocation: f,
			Mongodb:    f.GetMongoDBSpec(),
			DBOptions: &framework.DBOptions{
				DBType:         framework.StandAlone,
				SslModeEnabled: false,
			},
		}
	})
	AfterEach(func() {
		By("Deleting everything")
		err := to.DeleteBackupConfiguration()
		Expect(err).NotTo(HaveOccurred())

		err = to.DeleteRepository()
		Expect(err).NotTo(HaveOccurred())

		err = to.DeleteMongoDB()
		Expect(err).NotTo(HaveOccurred())

		to.CheckIfEverythingIsCleaned().Should(Succeed())
	})

	var runner = func() {
		By("Creating MongoDB")
		err := to.CreateMongoDB()
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for MongoDB  to be ready")
		to.CheckReadinessOfMongoDB().Should(Succeed())

		By("Applying Repository")
		err = to.CreateRepository()
		Expect(err).NotTo(HaveOccurred())

		By("Applying Backup Configuration")
		err = to.CreateBackupConfiguration()
		Expect(err).NotTo(HaveOccurred())

		By("Inserting Data in the Database for backup")
		err = to.InsertData()
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for success of Backup Configuration")
		to.CheckSuccessOfBackupConfiguration().Should(Succeed())
	}

	var set = func() {
		to.Mongodb = to.GetMongoDBSpec(to.DBOptions)
	}

	XContext("for Sharded", func() {
		BeforeEach(func() {
			to.DBType = framework.Sharded
		})
		It("should succeeded", func() {
			set()
			runner()
		})
	})
	Context("for replica set", func() {
		BeforeEach(func() {
			to.DBType = framework.ReplicaSet
		})
		It("should succeeded", func() {
			set()
			runner()
		})
	})
	XContext("for standalone", func() {
		BeforeEach(func() {
			to.DBType = framework.StandAlone
		})
		It("should succeeded", func() {
			set()
			runner()
		})
	})
})
