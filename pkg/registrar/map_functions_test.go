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

package registrar_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/vmware-tanzu/cartographer/pkg/apis/v1alpha1"
	"github.com/vmware-tanzu/cartographer/pkg/registrar"
	"github.com/vmware-tanzu/cartographer/pkg/registrar/registrarfakes"
)

var _ = Describe("MapFunctions", func() {
	Describe("ClusterSupplyChainToWorkloadRequests", func() {
		var (
			clientObjects      []client.Object
			mapper             *registrar.Mapper
			fakeClientBuilder  *fake.ClientBuilder
			scheme             *runtime.Scheme
			fakeLogger         *registrarfakes.FakeLogger
			clusterSupplyChain client.Object
			result             []reconcile.Request
		)

		BeforeEach(func() {
			scheme = runtime.NewScheme()
			fakeClientBuilder = fake.NewClientBuilder()
			fakeLogger = &registrarfakes.FakeLogger{}

			clusterSupplyChain = &v1alpha1.ClusterSupplyChain{
				Spec: v1alpha1.SupplyChainSpec{
					Selector: map[string]string{
						"myLabel": "myLabelsValue",
					},
				},
			}
		})

		JustBeforeEach(func() {
			fakeClientBuilder.
				WithScheme(scheme).
				WithObjects(clientObjects...)

			fakeClient := fakeClientBuilder.Build()

			mapper = &registrar.Mapper{
				Client: fakeClient,
				Logger: fakeLogger,
			}

			result = mapper.ClusterSupplyChainToWorkloadRequests(clusterSupplyChain)
		})

		Context("client.List returns an error", func() {
			// By using a scheme without v1alpha1, the client will error when handed our Objects
			It("logs an error to the client", func() {
				Expect(result).To(BeEmpty())

				Expect(fakeLogger.ErrorCallCount()).To(Equal(1))
				firstArg, secondArg, _ := fakeLogger.ErrorArgsForCall(0)
				Expect(firstArg).NotTo(BeNil())
				Expect(secondArg).To(Equal("cluster supply chain to workload requests: client list"))
			})
		})

		Context("client does not return errors", func() {
			BeforeEach(func() {
				// By including the scheme, the client will not error when handed our Objects
				err := v1alpha1.AddToScheme(scheme)
				Expect(err).ToNot(HaveOccurred())
			})
			Context("no workloads", func() {
				BeforeEach(func() {
					clusterSupplyChain = &v1alpha1.ClusterSupplyChain{
						Spec: v1alpha1.SupplyChainSpec{
							Selector: map[string]string{
								"myLabel": "myLabelsValue",
							},
						},
					}
				})
				It("returns an empty list of requests", func() {
					Expect(result).To(BeEmpty())
				})
			})
			Context("workloads", func() {
				var workload *v1alpha1.Workload
				BeforeEach(func() {
					workload = &v1alpha1.Workload{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "first-workload",
							Namespace: "first-namespace",
						},
						TypeMeta: metav1.TypeMeta{
							Kind:       "Workload",
							APIVersion: "carto.run/v1alpha1",
						},
					}
				})

				Context("supply chain with one matching workload", func() {
					BeforeEach(func() {
						workload.Labels = map[string]string{
							"myLabel": "myLabelsValue",
						}
						clientObjects = []client.Object{workload}
					})

					It("returns an empty list of requests", func() {
						expected := []reconcile.Request{
							{
								types.NamespacedName{
									Namespace: "first-namespace",
									Name:      "first-workload",
								},
							},
						}

						Expect(result).To(Equal(expected))
					})
				})
				Context("supply chain without matching workload", func() {
					BeforeEach(func() {
						workload.Labels = map[string]string{
							"myLabel": "otherLabel",
						}
						clientObjects = []client.Object{workload}
					})
					It("returns an empty list of requests", func() {
						Expect(result).To(BeEmpty())
					})
				})
			})

			Context("when function is passed an object that is not a supplyChain", func() {
				BeforeEach(func() {
					clusterSupplyChain = &v1alpha1.Workload{}
				})
				It("logs a helpful error", func() {
					Expect(result).To(BeEmpty())

					Expect(fakeLogger.ErrorCallCount()).To(Equal(1))
					firstArg, secondArg, _ := fakeLogger.ErrorArgsForCall(0)
					Expect(firstArg).To(BeNil())
					Expect(secondArg).To(Equal("cluster supply chain to workload requests: cast to ClusterSupplyChain failed"))
				})
			})
		})
	})
})