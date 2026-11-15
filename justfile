controller_gen := "sigs.k8s.io/controller-tools/cmd/controller-gen@latest"

gen:
    go run {{controller_gen}} \
        object:headerFile="" paths=./api/... \
        crd paths=./api/... output:crd:dir=config/crd \
        rbac:roleName=scale0-operator paths=./internal/... output:rbac:dir=config/rbac
