func (a {{.AuthName}}Auth) Can{{.Action}}(ctx context.Context, r *ent.{{.AuthName}}) bool {
	can, err := a.authSvc.Enforcer.Enforce(a.identityFn(ctx), r, "{{.AuthNameLower}}::{{.ActionLower}}")
	if err != nil {
		logx.WithContext(ctx).Errorf("enforce {{.AuthNameLower}}::{{.ActionLower}} fail: %v", err)
		return false
	}
	return can
}