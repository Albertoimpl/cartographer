// Copyright 2021 VMware
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package templates_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/valyala/fasttemplate"
	. "github.com/vmware-tanzu/cartographer/pkg/utils/matchers"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/vmware-tanzu/cartographer/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/cartographer/pkg/eval"
	"github.com/vmware-tanzu/cartographer/pkg/templates"
	"github.com/vmware-tanzu/cartographer/pkg/templates/templatesfakes"
)

var _ = Describe("Interpolator", func() {
	Describe("InterpolateLeafNode Stubbing executor", func() {
		var (
			template        []byte
			tagInterpolator templatesfakes.FakeTagInterpolator
			executor        templates.TemplateExecutor
		)

		BeforeEach(func() {
			template = []byte("some-template")
		})

		Context("when executor returns an error", func() {
			BeforeEach(func() {
				executor = func(template, startTag, endTag string, f fasttemplate.TagFunc) (string, error) {
					return "", fmt.Errorf("some error")
				}
			})

			It("returns an error", func() {
				_, err := templates.InterpolateLeafNode(executor, template, &tagInterpolator)
				Expect(err).To(BeMeaningful("interpolate tag:"))
			})
		})

		Context("when executor returns a stable result", func() {
			BeforeEach(func() {
				executor = func(template, startTag, endTag string, f fasttemplate.TagFunc) (string, error) {
					return "some result", nil
				}
			})

			It("returns the result as a byte array", func() {
				result, err := templates.InterpolateLeafNode(executor, template, &tagInterpolator)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("some result"))
			})
		})
	})

	Describe("InterpolateLeafNode with fasttemplate and Stubbing TagInterpolator", func() {
		var (
			template        []byte
			tagInterpolator templatesfakes.FakeTagInterpolator
		)

		BeforeEach(func() {
			tagInterpolator = templatesfakes.FakeTagInterpolator{}
			template = []byte("some-template-with-$(some-tag)$")
		})

		// Todo: Missing happy path?

		Context("When the tag interpolator returns an error", func() {
			BeforeEach(func() {
				tagInterpolator.InterpolateTagReturns(0, fmt.Errorf("some error"))
			})
			It("returns an error", func() {
				_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, &tagInterpolator)
				Expect(err).To(BeMeaningful("interpolate tag: "))
			})
		})
	})

	Describe("InterpolateLeafNode with fasttemplate and StandardTagInterpolator and eval.Evaluator", func() {
		var (
			workload        *v1alpha1.Workload
			template        []byte
			params          []templates.Param
			sources         []templates.SourceInput
			images          []templates.ImageInput
			configs         []templates.ConfigInput
			tagInterpolator templates.StandardTagInterpolator
		)

		BeforeEach(func() {
			workload = &v1alpha1.Workload{}
			params = []templates.Param{
				{
					Name:  "an amazing param",
					Value: apiextensionsv1.JSON{Raw: []byte(`"exactly what you want"`)},
				},
				{
					Name:  "another_param",
					Value: apiextensionsv1.JSON{Raw: []byte(`"everything you need"`)},
				},
			}
			sources = []templates.SourceInput{
				{
					Name:     "first_source",
					URL:      "https://example.com/first",
					Revision: "c001cafe",
				},
				{
					Name:     "second_source",
					URL:      "https://example.com/second",
					Revision: "c0c0a",
				},
			}
			images = []templates.ImageInput{
				{
					Name:  "first_image",
					Image: "some://image/place",
				},
				{
					Name:  "second_image",
					Image: "some://other/image/place",
				},
			}
			configs = []templates.ConfigInput{
				{
					Name:   "first_config",
					Config: "kittens are furry",
				},
				{
					Name:   "second_config",
					Config: "branston pickles are great",
				},
			}
			tagInterpolator = templates.StandardTagInterpolator{
				Evaluator: eval.EvaluatorBuilder(),
			}
			tagInterpolator.Context = templates.StampContext{Workload: workload, Params: params, Sources: sources, Images: images, Configs: configs}
		})

		Context("given a template with no tags to interpolate", func() {
			BeforeEach(func() {
				template = []byte("hello, this is dog")
			})

			It("returns the same byte array", func() {
				returnedInterpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
				Expect(err).NotTo(HaveOccurred())
				Expect(returnedInterpolatedTemplate).To(Equal(string(template)))
			})
		})

		Context("given a template with an empty tag", func() {
			BeforeEach(func() {
				template = []byte("Look at this empty tag ---> $()$")
			})

			It("Returns an error explaining that empty jsonpath is not allowed", func() {
				_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
				Expect(err).To(BeMeaningful("interpolate tag: "))
				Expect(err).To(BeMeaningful("empty jsonpath not allowed"))
			})
		})

		Context("given a template with a tag for an unknown field in the stamp context", func() {
			BeforeEach(func() {
				template = []byte("I've never heard of a $(snarfblatt.name)$")
			})

			It("Returns an error that something went wrong in evaluating jsonpath", func() {
				_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
				Expect(err).To(BeMeaningful("interpolate tag: "))
				Expect(err).To(BeMeaningful("evaluate jsonpath: "))
			})
		})

		Context("given a template with a tag for an unknown subfield in the stamp context", func() {
			BeforeEach(func() {
				template = []byte("Workloads don't have $(workload.vacationLoad)$")
			})

			It("Returns an error that something went wrong in evaluating jsonpath", func() {
				_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
				Expect(err).To(BeMeaningful("interpolate tag: "))
				Expect(err).To(BeMeaningful("evaluate jsonpath: "))
			})
		})

		Context("given a stampContext with some values defined", func() {
			BeforeEach(func() {
				workloadName := "work-name"
				workloadNamespace := "work-namespace"

				workload.Name = workloadName
				workload.Namespace = workloadNamespace
			})

			Context("and a tag pointing to a field that is empty", func() {
				BeforeEach(func() {
					template = []byte("this workload does not have an env: $(workload.spec.source)$ <-- so this shouldn't work")
				})

				It("Returns an error that a tag points to a nil value", func() {
					_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
					Expect(err).To(BeMeaningful("interpolate tag: "))
					Expect(err).To(BeMeaningful("tag must not point to nil value: workload.spec.source"))
				})
			})

			Context("and a tag pointing to a string field that can be interpolated", func() {
				BeforeEach(func() {
					template = []byte("this is the place to put the name: $(workload.metadata.name)$ <-- see it there?")
				})

				It("returns the proper string", func() {
					interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

					Expect(err).NotTo(HaveOccurred())
					Expect(interpolatedTemplate).To(Equal("this is the place to put the name: work-name <-- see it there?"))
				})
			})

			Context("where there are multiple tags", func() {
				BeforeEach(func() {
					template = []byte("this is the place to put the name: $(workload.metadata.name)$ and the namespace: $(workload.metadata.namespace)$")
				})

				It("returns the proper string", func() {
					interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

					Expect(err).NotTo(HaveOccurred())
					Expect(interpolatedTemplate).To(Equal("this is the place to put the name: work-name and the namespace: work-namespace"))
				})
			})

			Context("and a tag that refers to a list", func() {
				BeforeEach(func() {
					workload.Spec.Env = []corev1.EnvVar{
						{
							Name:  "George",
							Value: "Carver",
						},
					}
					template = []byte("this is the place to put the env: $(workload.spec.env)$ <-- see it there?")
				})

				It("returns the proper string", func() {
					interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

					Expect(err).NotTo(HaveOccurred())
					Expect(interpolatedTemplate).To(Equal(
						`this is the place to put the env: [{"name":"George","value":"Carver"}] <-- see it there?`,
					))
				})
			})

			Context("and a tag that refers to an object", func() {
				BeforeEach(func() {
					url := "some.great-repo.com"
					branch := "main"

					workload.Spec.Source = &v1alpha1.WorkloadSource{
						Git: &v1alpha1.WorkloadGit{
							URL: &url,
							Ref: &v1alpha1.WorkloadGitRef{
								Branch: &branch,
							},
						},
					}

					template = []byte("this is the place to put the source: $(workload.spec.source.git)$")

				})

				It("returns the proper string", func() {
					interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

					Expect(err).NotTo(HaveOccurred())
					Expect(interpolatedTemplate).To(Equal(
						`this is the place to put the source: {"ref":{"branch":"main"},"url":"some.great-repo.com"}`,
					))

				})

			})

			Context("and a tag that refers to an item in a list", func() {
				BeforeEach(func() {
					workload.Spec.ServiceClaims = []v1alpha1.WorkloadServiceClaim{
						{
							Name: "a-service",
							Ref: &v1alpha1.WorkloadServiceClaimReference{
								APIVersion: "someApi",
								Kind:       "some-kind",
								Name:       "my-service",
							},
						},
					}
				})

				Context("by index", func() {
					BeforeEach(func() {
						template = []byte("this is the item: $(workload.spec.serviceClaims[0].ref.kind)$")
					})

					It("returns the proper string", func() {
						interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

						Expect(err).NotTo(HaveOccurred())
						Expect(interpolatedTemplate).To(Equal(
							"this is the item: some-kind",
						))
					})
				})

				Context("by an expression", func() {
					Context("which is properly formatted", func() {
						BeforeEach(func() {
							template = []byte(`here is a kind: $(workload.spec.serviceClaims[?(@.name=="a-service")].ref.kind)$`)
						})

						It("returns the proper string", func() {
							interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

							Expect(err).NotTo(HaveOccurred())
							Expect(interpolatedTemplate).To(Equal(
								"here is a kind: some-kind",
							))
						})
					})

					Context("which matches multiple objects", func() {
						BeforeEach(func() {
							template = []byte(`here is a kind: $(workload.spec.serviceClaims[?(@.ref.apiVersion=="someApi")].ref.kind)$`)

							workload.Spec.ServiceClaims = append(workload.Spec.ServiceClaims, v1alpha1.WorkloadServiceClaim{
								Name: "another-service",
								Ref: &v1alpha1.WorkloadServiceClaimReference{
									APIVersion: "someApi",
									Kind:       "another-kind",
									Name:       "another-service",
								},
							})
						})

						It("returns an error that there are too many results", func() {
							_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
							Expect(err).To(BeMeaningful("too many results for the query: "))
						})
					})
				})
			})

			Context("when tag refers to multiple objects in the spec", func() {
				BeforeEach(func() {
					template = []byte("this will be a lot: $(workload.spec.serviceClaims)$")
					workload.Spec.ServiceClaims = []v1alpha1.WorkloadServiceClaim{
						{
							Name: "a-service",
							Ref: &v1alpha1.WorkloadServiceClaimReference{
								APIVersion: "someApi",
								Kind:       "some-kind",
								Name:       "my-service",
							},
						},
						{
							Name: "another-service",
							Ref: &v1alpha1.WorkloadServiceClaimReference{
								APIVersion: "another-api",
								Kind:       "another-kind",
								Name:       "another-service",
							},
						},
					}
				})

				It("returns the proper string", func() {
					interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

					Expect(err).NotTo(HaveOccurred())
					Expect(interpolatedTemplate).To(Equal(
						`this will be a lot: [{"name":"a-service","ref":{"apiVersion":"someApi","kind":"some-kind","name":"my-service"}},{"name":"another-service","ref":{"apiVersion":"another-api","kind":"another-kind","name":"another-service"}}]`,
					))
				})
			})

			Context("when path is to some object that serializes with extra quotes", func() {
				BeforeEach(func() {
					template = []byte("quantities are weird: $(workload.spec.resources.requests.cpu)$")

					workload.Spec.Resources = &corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU: resource.MustParse("10Gi"),
						},
					}
				})

				It("returns the proper string", func() {
					interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

					Expect(err).NotTo(HaveOccurred())
					Expect(interpolatedTemplate).To(Equal(
						"quantities are weird: 10Gi",
					))
				})
			})
		})

		Context("given a template with a tag for a param", func() {
			Context("that is not in the stampcontext", func() {
				BeforeEach(func() {
					template = []byte("in an empty input, you won't find $(params[0].value)$")
				})

				It("Returns an error that it cannot evaluate the path", func() {
					_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
					Expect(err).To(BeMeaningful("interpolate tag: "))
					Expect(err).To(BeMeaningful("evaluate jsonpath: "))
				})
			})

			Context("that is in the stampcontext", func() {
				Context("where tag uses array indexing", func() {
					BeforeEach(func() {
						template = []byte("with the populated input, you can find $(params[1].value)$")
					})

					It("returns the proper string", func() {
						interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

						Expect(err).NotTo(HaveOccurred())
						Expect(interpolatedTemplate).To(Equal(
							"with the populated input, you can find everything you need",
						))
					})
				})

				Context("where tag uses expression matching", func() {
					BeforeEach(func() {
						template = []byte(`with the populated input, you can find $(params[?(@.name=="an amazing param")].value)$`)
					})


					It("returns the proper string", func() {
						interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

						Expect(err).NotTo(HaveOccurred())
						Expect(interpolatedTemplate).To(Equal(
							"with the populated input, you can find exactly what you want",
						))
					})
				})
			})
		})

		Context("given a template with a tag for a source", func() {
			Context("that is not in the input", func() {
				BeforeEach(func() {
					template = []byte("in an empty input, you won't find $(sources[0].url)$")
				})

				It("Returns an error that it cannot evaluate the path", func() {
					_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
					Expect(err).To(BeMeaningful("interpolate tag: "))
					Expect(err).To(BeMeaningful("evaluate jsonpath: "))
				})
			})

			Context("that is in the input", func() {
				Context("where tag uses array indexing", func() {
					BeforeEach(func() {
						template = []byte("with the populated source, you can find $(sources[1].url)$ and $(sources[1].revision)$")
					})

					It("returns the proper string", func() {
						interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

						Expect(err).NotTo(HaveOccurred())
						Expect(interpolatedTemplate).To(Equal(
							"with the populated source, you can find https://example.com/second and c0c0a",
						))
					})
				})

				Context("where tag uses expression matching", func() {
					BeforeEach(func() {
						template = []byte(`with the populated source, you can find $(sources[?(@.name=="first_source")].url)$ and $(sources[?(@.name=="second_source")].revision)$`)
					})

					It("returns the proper string", func() {
						interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

						Expect(err).NotTo(HaveOccurred())
						Expect(interpolatedTemplate).To(Equal(
							"with the populated source, you can find https://example.com/first and c0c0a",
						))
					})
				})

				Context("where tag accesses an unknown key", func() {
					BeforeEach(func() {
						template = []byte("with the populated source, you can't find $(sources[1].qwijibo)$ because it does not exist")
					})

					It("Returns an error that it cannot find the value qwijibo", func() {
						_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
						Expect(err).To(BeMeaningful("interpolate tag: "))
						Expect(err).To(BeMeaningful("evaluate jsonpath: evaluate: find results: qwijibo is not found"))
					})
				})
			})
		})

		Context("given a template with a tag for an image", func() {
			Context("that is not in the input", func() {
				BeforeEach(func() {
					template = []byte("in an empty input, you won't find $(images[0].image)$")
				})

				It("Returns an error that it cannot evaluate the path", func() {
					_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
					Expect(err).To(BeMeaningful("interpolate tag: "))
					Expect(err).To(BeMeaningful("evaluate jsonpath: "))
				})
			})

			Context("that is in the input", func() {
				Context("where tag uses array indexing", func() {
					BeforeEach(func() {
						template = []byte("with the populated image, you can find $(images[1].image)$")
					})

					It("returns the proper string", func() {
						interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

						Expect(err).NotTo(HaveOccurred())
						Expect(interpolatedTemplate).To(Equal(
							"with the populated image, you can find some://other/image/place",
						))
					})
				})

				Context("where tag uses expression matching", func() {
					BeforeEach(func() {
						template = []byte(`with the populated image, you can find $(images[?(@.name=="first_image")].image)$`)
					})


					It("returns the proper string", func() {
						interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

						Expect(err).NotTo(HaveOccurred())
						Expect(interpolatedTemplate).To(Equal(
							"with the populated image, you can find some://image/place",
						))
					})
				})

				Context("where tag accesses an unknown key", func() {
					BeforeEach(func() {
						template = []byte("with the populated image, you can't find $(images[1].qwijibo)$ because it does not exist")
					})

					It("Returns an error that it cannot evaluate the value qwijibo", func() {
						_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
						Expect(err).To(BeMeaningful("interpolate tag: "))
						Expect(err).To(BeMeaningful("evaluate jsonpath: evaluate: find results: qwijibo is not found"))
					})
				})
			})
		})

		Context("given a template with a tag for an config", func() {
			Context("that is not in the input", func() {
				BeforeEach(func() {
					template = []byte("in an empty input, you won't find $(configs[0].config)$")
				})

				It("Returns an error that it cannot evaluate the path", func() {
					_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
					Expect(err).To(BeMeaningful("interpolate tag: "))
					Expect(err).To(BeMeaningful("evaluate jsonpath: "))
				})
			})

			Context("that is in the input", func() {
				Context("where tag uses array indexing", func() {
					BeforeEach(func() {
						template = []byte("with the populated config, you can find $(configs[1].config)$")
					})

					It("returns the proper string", func() {
						interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

						Expect(err).NotTo(HaveOccurred())
						Expect(interpolatedTemplate).To(Equal(
							"with the populated config, you can find branston pickles are great",
						))
					})
				})

				Context("where tag uses expression matching", func() {
					BeforeEach(func() {
						template = []byte(`with the populated config, you can find $(configs[?(@.name=="first_config")].config)$`)
					})

					It("returns the proper string", func() {
						interpolatedTemplate, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)

						Expect(err).NotTo(HaveOccurred())
						Expect(interpolatedTemplate).To(Equal(
							"with the populated config, you can find kittens are furry",
						))
					})
				})

				Context("where tag accesses an unknown key", func() {
					BeforeEach(func() {
						template = []byte("with the populated config, you can't find $(configs[1].qwijibo)$ because it does not exist")
					})

					It("Returns an error that it cannot evaluate the path", func() {
						_, err := templates.InterpolateLeafNode(fasttemplate.ExecuteFuncStringWithErr, template, tagInterpolator)
						Expect(err).To(BeMeaningful("interpolate tag: "))
						Expect(err).To(BeMeaningful("evaluate jsonpath: "))
					})
				})
			})
		})
	})
})

var _ = Describe("StandardTagInterpolator", func() {
	var (
		err                     error
		context                 templates.StampContext
		evaluator               templatesfakes.FakeEvaluator
		standardTagInterpolator templates.StandardTagInterpolator
	)

	ItDoesNotReturnAnError := func() {
		It("does not return an error", func() {
			Expect(err).ShouldNot(HaveOccurred())
		})
	}

	ItReturnsAHelpfulError := func(expectedErrorSubstring string) {
		It("returns a helpful error", func() {
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedErrorSubstring))
		})
	}

	BeforeEach(func() {
		evaluator = templatesfakes.FakeEvaluator{}

		standardTagInterpolator = templates.StandardTagInterpolator{
			Context:   context,
			Evaluator: &evaluator,
		}
	})

	Describe("InterpolateTag", func() {
		var (
			writer   templatesfakes.FakeWriter
			tag      string
			writeLen int
		)

		JustBeforeEach(func() {
			writeLen, err = standardTagInterpolator.InterpolateTag(&writer, tag)
		})

		Context("with a tag", func() {
			BeforeEach(func() {
				tag = "some tag"
			})

			Context("when the evaluator returns an error", func() {
				BeforeEach(func() {
					evaluator.EvaluateJsonPathReturns("", fmt.Errorf("some error"))
				})

				ItReturnsAHelpfulError("evaluate jsonpath: ")
			})

			Context("when the evaluator returns a nil value", func() {
				BeforeEach(func() {
					evaluator.EvaluateJsonPathReturns(nil, nil)
				})

				ItReturnsAHelpfulError("tag must not point to nil value: ")
			})

			Context("when the evaluator returns a string", func() {
				var mockWriteLen int
				BeforeEach(func() {
					writer = templatesfakes.FakeWriter{}
					mockWriteLen = 123
					writer.WriteReturns(mockWriteLen, nil)

					evaluator.EvaluateJsonPathReturns("some value", nil)
				})

				It("calls the writer with that value", func() {
					byteArray := writer.WriteArgsForCall(0)
					Expect(byteArray).To(Equal([]byte("some value")))
				})

				ItDoesNotReturnAnError()

				It("passes back the length from the writer", func() {
					Expect(writeLen).To(Equal(mockWriteLen))
				})

				Context("and the writer fails to write", func() {
					BeforeEach(func() {
						writer.WriteReturns(0, fmt.Errorf("some error"))
					})

					ItReturnsAHelpfulError("writer write: ")
				})
			})

			Context("when the evaluator returns a non string object", func() {
				var mockWriteLen int
				BeforeEach(func() {
					writer = templatesfakes.FakeWriter{}
					mockWriteLen = 123
					writer.WriteReturns(mockWriteLen, nil)

					evaluator.EvaluateJsonPathReturns(3, nil)
				})

				It("calls the writer with a json representation of the object", func() {
					byteArray := writer.WriteArgsForCall(0)
					Expect(byteArray).To(Equal([]byte("3")))
				})

				It("passes back the length from the writer", func() {
					Expect(writeLen).To(Equal(mockWriteLen))
				})

				ItDoesNotReturnAnError()
			})
		})
	})
})
