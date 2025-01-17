/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lifecycle_test

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/karpenter-core/pkg/apis/v1alpha5"
	"github.com/aws/karpenter-core/pkg/apis/v1beta1"
	"github.com/aws/karpenter-core/pkg/cloudprovider"
	"github.com/aws/karpenter-core/pkg/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/aws/karpenter-core/pkg/test/expectations"
)

var _ = Describe("NodeClaim/Launch", func() {
	var nodePool *v1beta1.NodePool
	BeforeEach(func() {
		nodePool = test.NodePool()
	})
	It("should launch an instance when a new Machine is created", func() {
		machine := test.NodeClaim(v1beta1.NodeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					v1beta1.NodePoolLabelKey: nodePool.Name,
				},
			},
		})
		ExpectApplied(ctx, env.Client, nodePool, machine)
		ExpectReconcileSucceeded(ctx, nodeClaimController, client.ObjectKeyFromObject(machine))

		machine = ExpectExists(ctx, env.Client, machine)

		Expect(cloudProvider.CreateCalls).To(HaveLen(1))
		Expect(cloudProvider.CreatedNodeClaims).To(HaveLen(1))
		_, err := cloudProvider.Get(ctx, machine.Status.ProviderID)
		Expect(err).ToNot(HaveOccurred())
	})
	It("should add the MachineLaunched status condition after creating the Machine", func() {
		machine := test.NodeClaim(v1beta1.NodeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					v1alpha5.ProvisionerNameLabelKey: nodePool.Name,
				},
			},
		})
		ExpectApplied(ctx, env.Client, nodePool, machine)
		ExpectReconcileSucceeded(ctx, nodeClaimController, client.ObjectKeyFromObject(machine))

		machine = ExpectExists(ctx, env.Client, machine)
		Expect(ExpectStatusConditionExists(machine, v1beta1.NodeLaunched).Status).To(Equal(v1.ConditionTrue))
	})
	It("should delete the machine if InsufficientCapacity is returned from the cloudprovider", func() {
		cloudProvider.NextCreateErr = cloudprovider.NewInsufficientCapacityError(fmt.Errorf("all instance types were unavailable"))
		machine := test.NodeClaim()
		ExpectApplied(ctx, env.Client, machine)
		ExpectReconcileSucceeded(ctx, nodeClaimController, client.ObjectKeyFromObject(machine))
		ExpectFinalizersRemoved(ctx, env.Client, machine)
		ExpectNotFound(ctx, env.Client, machine)
	})
})
