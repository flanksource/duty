UPDATE casbin_rule SET v3 = 'allow' WHERE ptype = 'p' and (v3 is null or v3 = '')
