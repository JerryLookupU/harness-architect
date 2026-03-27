package svc

type ServiceContext struct {
	Root string
}

func NewServiceContext(root string) *ServiceContext {
	return &ServiceContext{Root: root}
}
