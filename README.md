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
// to prepare the cluster for the upgrade and validate the rollout,
// then to validate or postpone the successive pods updates.
type STSRolloutHooks interface {
	// Name returns the hook's name
	Name() string

	// PodUpdateTransition is called between pods updates.
	// prev is the previously updated pod (or nil when we're starting a new rollout).
	// next is the pod we're about to update (or nil when we just updated the last pod).
	// If PodUpdateTransition returns true, the controller will proceed updating the next pod.
	// If PodUpdateTransition returns false, the controller will postpone the update, and
	// will call PodUpdateTransition again later, until it succeed.
	PodUpdateTransition(logger logr.Logger, sts *appsv1.StatefulSet, prev, next *v1.Pod) bool
}
```

They must then be registered from factory.go `init()`:
```Go
  import "github.com/bpineau/statefulset-pilot/pkg/hooks/myhook"
  ...
  Register("myhook", myhook.New)
```

