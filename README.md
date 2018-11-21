# statefulset-pilot

/!\ WARNING this is EXPERIMENTAL

**statefulset-pilot** calls hooks between each pod update during a statefulset rollout, and before the rollout itself.

When such an hook returns false, the next pod update is temporarly postponed,
and the hook will be called again later, until it succeed (returns true).

This gives the opportunity to implement some extras, external readiness checks,
or some apps registration/cleanup/... steps on freshly updated pods before resuming.

For instance, after an Elasticsearch pod update, a hook may throttle the rollout until the pod
catchs up replicating indexes data.


## Subscribing hooks

Statefulsets are subscribed to a statefulset-pilot hook with  the `dd-statefulset-pilot: hook-name` label.
For instance, to register an elasticsearch statefulset to the `elasticsearch` statefulset-pilot hook,
we would set the `dd-statefulset-pilot: elasticsearch` label.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: es-cluster
  labels:
    dd-statefulset-pilot: elasticsearch
spec:
  replicas: 3
  updateStrategy:
    rollingUpdate:
      partition: 3  # set that == replicas
    type: RollingUpdate
```

Pod without this `statefulset-pilot` label are ignored by statefulset-pilot controller.


## Writing hooks

Hooks are Go code implementing the following interface:


```Go
// STSRolloutHooks is used between statefulset rollout transitions,
// to prepare cluster for the upgrade and validate the rollout,
// then to validate or postpone successive pods updates.
type STSRolloutHooks interface {
	// Name returns the hook's name
	Name() string

	// BeforeSTSRollout is called when a rollout is needed.
	// If this function returns true, the rollout will start right away.
	// Otherwise, BeforeSTSRollout will be called again later.
	BeforeSTSRollout(sts *appsv1.StatefulSet) bool

	// BeforePodUpdate is called after a pod had been updated,
	// and before we update the next pod. If it returns false,
	// we postpone updating the next pod, and will call
	// BeforePodUpdate again later, until we're ready to continue.
	// The pod passed in argument is the pod we just updated.
	// This is NOT called before we update the first statefulset
	// pod (use BeforeSTSRollout() to control the rollout start).
	BeforePodUpdate(pod *v1.Pod) bool
}
```

They must then be registered from factory.go `init()`:
```Go
  import "github.com/bpineau/statefulset-pilot/pkg/hooks/myhook"
  ...
  Register("myhook", myhook.New)
```

