# statefulset-pilot

/!\ WARNING this is an EXPERIMENTAL PROTOTYPE

**statefulset-pilot** calls validating hooks between each pod update during a statefulset rollout.

Hooks can return false to temporarily postpone the next pod update (they'll be called again later).
They are provided the statefulset, previous and next pods structs, so they can also execute pre and
post update tasks.

For instance an Elasticsearch hook may flush sync and disable realocations before a pod update,
and re-enable realocations then wait for the pod to become green after the update.


## Subscribing satefulset to the pilot

Statefulsets are subscribed to a statefulset-pilot hook with the `dd-statefulset-pilot: hook-name` label.

For instance, to register an elasticsearch statefulset to the `elasticsearch` statefulset-pilot hook,
we would set the `dd-statefulset-pilot: elasticsearch` label.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: es-cluster
  labels:
    # this subscribe this sts to the statefulset-pilot's elasticsearch hook
    dd-statefulset-pilot: elasticsearch
spec:
  replicas: 3
  updateStrategy:
    rollingUpdate:
      # set partition == replicas by default, so rollouts will wait for the controller
      partition: 3
    type: RollingUpdate
```

Pod without this `statefulset-pilot` label are ignored by statefulset-pilot controller.


## Writing hooks

Hooks are Go code implementing the following interface:

```Go
// STSRolloutHooks is called between statefulset pods updates.
// The `next` pod is updated when the hook returns true.
type STSRolloutHooks interface {
        // Name returns the hook's name
        Name() string

        // PodUpdateTransition is called between pods updates.
        // prev is the previously updated pod (or nil when we're starting a new rollout).
        // next is the pod we're about to update (or nil after we updated the last pod).
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

