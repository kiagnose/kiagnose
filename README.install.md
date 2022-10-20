# Kiagnose Installation

In order to install Kiagnose on your namespace:

1. Clone this repository:

```bash
git clone https://github.com/kiagnose/kiagnose.git
```

2. Apply the following manifest from the project root directory to the desired target namespace:

```bash
kubectl apply -f ./manifests/kiagnose.yaml -n <target-namespace>
```

This manifest contains the following objects:

- `kiagnose` [ServiceAccount](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#service-account-permissions)
  object.
- `kiagnose` [Role](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole)
  and [RoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding)
  objects.

# Kiagnose Removal

1. Delete the objects contained in the following manifest from the project root directory:

```bash
kubectl delete -f ./manifests/kiagnose.yaml  -n <target-namespace>
```
