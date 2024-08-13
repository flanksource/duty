package types

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = Describe("EnvVar", func() {
	Describe("Scan", func() {
		Context("with static value", func() {
			It("should scan static value correctly", func() {
				var envVar EnvVar
				err := envVar.Scan("foo")
				Expect(err).To(BeNil())
				Expect(envVar.ValueStatic).To(Equal("foo"))
			})
		})

		Context("with configmap value", func() {
			It("should scan configmap value correctly", func() {
				var envVar EnvVar
				err := envVar.Scan("configmap://foo/bar")
				Expect(err).To(BeNil())
				Expect(envVar.ValueFrom.ConfigMapKeyRef.Name).To(Equal("foo"))
				Expect(envVar.ValueFrom.ConfigMapKeyRef.Key).To(Equal("bar"))
			})
		})

		Context("with secret value", func() {
			It("should scan secret value correctly", func() {
				var envVar EnvVar
				err := envVar.Scan("secret://foo/bar")
				Expect(err).To(BeNil())
				Expect(envVar.ValueFrom.SecretKeyRef.Name).To(Equal("foo"))
				Expect(envVar.ValueFrom.SecretKeyRef.Key).To(Equal("bar"))
			})
		})

		Context("with service account reference", func() {
			It("should scan valid service account reference", func() {
				var envVar EnvVar
				err := envVar.Scan("serviceaccount://my-service-account")
				Expect(err).To(BeNil())
				Expect(envVar.ValueFrom.ServiceAccount).To(Equal(lo.ToPtr("my-service-account")))
			})

			It("should return error for invalid service account reference format", func() {
				var envVar EnvVar
				err := envVar.Scan("serviceaccount://")
				Expect(err).To(HaveOccurred())
			})

			It("should return error for invalid service account reference name", func() {
				var envVar EnvVar
				err := envVar.Scan("serviceaccount:///invalid-name")
				Expect(err).To(HaveOccurred())
			})

			It("should return error for non-service account reference prefix", func() {
				var envVar EnvVar
				err := envVar.Scan("configmap://my-configmap")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with helm reference", func() {
			It("should scan valid helm reference", func() {
				var envVar EnvVar
				err := envVar.Scan("helm://canary-checker/the-key")
				Expect(err).To(BeNil())
				Expect(envVar.ValueFrom.HelmRef.Name).To(Equal("canary-checker"))
				Expect(envVar.ValueFrom.HelmRef.Key).To(Equal("the-key"))
			})

			It("should return error for invalid helm reference", func() {
				var envVar EnvVar
				err := envVar.Scan("helm:///canary-checker/the-key")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with invalid input", func() {
			It("should return error for non-string type", func() {
				var envVar EnvVar
				err := envVar.Scan(123)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("IsEmpty", func() {
		type fields struct {
			Name        string
			ValueStatic string
			ValueFrom   *EnvVarSource
		}

		tests := []struct {
			name   string
			fields fields
			want   bool
		}{
			{
				name: "nil",
				fields: fields{
					Name:        "",
					ValueStatic: "",
					ValueFrom:   nil,
				},
				want: true,
			},
			{
				name: "ValueStatic",
				fields: fields{
					Name:        "",
					ValueStatic: "ValueStatic",
					ValueFrom:   nil,
				},
				want: false,
			},
			{
				name: "non nil ValueFrom",
				fields: fields{
					Name:        "",
					ValueStatic: "",
					ValueFrom:   &EnvVarSource{},
				},
				want: true,
			},
			{
				name: "non nil ValueFrom",
				fields: fields{
					Name:        "",
					ValueStatic: "",
					ValueFrom: &EnvVarSource{
						ServiceAccount: lo.ToPtr(""),
						SecretKeyRef: &SecretKeySelector{
							Key: "",
						},
					},
				},
				want: true,
			},
		}

		for _, tt := range tests {
			It(tt.name, func() {
				e := EnvVar{
					Name:        tt.fields.Name,
					ValueStatic: tt.fields.ValueStatic,
					ValueFrom:   tt.fields.ValueFrom,
				}
				Expect(e.IsEmpty()).To(Equal(tt.want))
			})
		}
	})
})
