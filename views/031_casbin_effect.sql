UPDATE casbin_rule SET v3 = 'allow' WHERE ptype = 'p' and (v3 is null or v3 = '');

UPDATE casbin_rule SET v4 = 'true' WHERE ptype = 'p' and (v4 is null OR v4 = '');

UPDATE casbin_rule SET v5 = 'na' WHERE ptype = 'p' and (v5 is null OR v5 = '');
