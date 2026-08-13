package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/functions"
	_ "github.com/lib/pq"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("PostgREST role bootstrap", func() {
	versions := []struct {
		name    string
		version embeddedpostgres.PostgresVersion
	}{
		{name: "PostgreSQL 15", version: embeddedpostgres.V15},
		{name: "PostgreSQL 17", version: embeddedpostgres.V17},
	}

	for _, version := range versions {
		ginkgo.It("allows a CREATEROLE user to SET ROLE on "+version.name, ginkgo.Label("slow"), func() {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			Expect(err).NotTo(HaveOccurred())
			port := uint32(listener.Addr().(*net.TCPAddr).Port)
			Expect(listener.Close()).To(Succeed())

			postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
				Version(version.version).
				Port(port).
				Username("postgres").
				Password("postgres").
				Database("postgres").
				RuntimePath(filepath.Join(ginkgo.GinkgoT().TempDir(), "postgres")).
				StartTimeout(time.Minute).
				Logger(io.Discard))
			Expect(postgres.Start()).To(Succeed())
			defer func() {
				Expect(postgres.Stop()).To(Succeed())
			}()

			adminDB, err := sql.Open("postgres", fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", port))
			Expect(err).NotTo(HaveOccurred())
			defer adminDB.Close()

			_, err = adminDB.Exec(`
				CREATE ROLE duty_bootstrap LOGIN PASSWORD 'duty_bootstrap' CREATEROLE;
				GRANT CREATE ON DATABASE postgres TO duty_bootstrap;
				GRANT USAGE, CREATE ON SCHEMA public TO duty_bootstrap;
			`)
			Expect(err).NotTo(HaveOccurred())

			bootstrapURL := fmt.Sprintf("postgres://duty_bootstrap:duty_bootstrap@localhost:%d/postgres?sslmode=disable", port)
			bootstrapDB, err := sql.Open("postgres", bootstrapURL)
			Expect(err).NotTo(HaveOccurred())
			defer bootstrapDB.Close()

			scripts, err := functions.GetFunctions()
			Expect(err).NotTo(HaveOccurred())
			postgrestSQL, ok := scripts["postgrest.sql"]
			Expect(ok).To(BeTrue())
			config := api.Config{
				ConnectionString: bootstrapURL,
				Postgrest: api.PostgrestConfig{
					DBRole:     "postgrest_api",
					AnonDBRole: "postgrest_anon",
				},
			}

			for range 2 {
				_, err = bootstrapDB.Exec(postgrestSQL)
				Expect(err).NotTo(HaveOccurred())
				Expect(grantPostgrestRolesToCurrentUser(bootstrapDB, config)).To(Succeed())
			}

			conn, err := bootstrapDB.Conn(context.Background())
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close()
			for _, role := range []string{"postgrest_api", "postgrest_anon"} {
				_, err = conn.ExecContext(context.Background(), "SET ROLE "+role)
				Expect(err).NotTo(HaveOccurred())
				var currentRole string
				Expect(conn.QueryRowContext(context.Background(), "SELECT current_role").Scan(&currentRole)).To(Succeed())
				Expect(currentRole).To(Equal(role))
				_, err = conn.ExecContext(context.Background(), "RESET ROLE")
				Expect(err).NotTo(HaveOccurred())
			}

			if version.version == embeddedpostgres.V17 {
				for _, role := range []string{"postgrest_api", "postgrest_anon"} {
					var adminOption, setOption, inheritOption bool
					Expect(adminDB.QueryRow(`
						SELECT bool_or(admin_option), bool_or(set_option), bool_or(inherit_option)
						FROM pg_auth_members membership
						JOIN pg_roles granted_role ON granted_role.oid = membership.roleid
						JOIN pg_roles member_role ON member_role.oid = membership.member
						WHERE granted_role.rolname = $1 AND member_role.rolname = 'duty_bootstrap'
					`, role).Scan(&adminOption, &setOption, &inheritOption)).To(Succeed())
					Expect(adminOption).To(BeTrue())
					Expect(setOption).To(BeTrue())
					Expect(inheritOption).To(BeFalse())
				}
			}

			_, err = adminDB.Exec(postgrestSQL)
			Expect(err).NotTo(HaveOccurred())
			var superuserMemberships int
			Expect(adminDB.QueryRow(`
				SELECT count(*)
				FROM pg_auth_members membership
				JOIN pg_roles granted_role ON granted_role.oid = membership.roleid
				JOIN pg_roles member_role ON member_role.oid = membership.member
				WHERE granted_role.rolname IN ('postgrest_api', 'postgrest_anon')
				  AND member_role.rolname = 'postgres'
			`).Scan(&superuserMemberships)).To(Succeed())
			Expect(superuserMemberships).To(BeZero())
		})
	}
})
