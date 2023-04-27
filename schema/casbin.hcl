table "casbin_rule" {
  schema = schema.public
  column "id" {
    null    = false
    type    = bigserial
    unsigned = true
  }
  column "ptype" {
    null = false
    type = varchar(100)
  }
  column "v0" {
    null    = false
    type = varchar(100)
  }
  column "v1" {
    null    = false
    type = varchar(100)
  }
  column "v2" {
    null    = true
    type = varchar(100)
  }
  column "v3" {
    null    = true
    type = varchar(100)
  }
  column "v4" {
    null    = true
    type = varchar(100)
  }
  column "v5" {
    null    = true
    type = varchar(100)
  }

  primary_key {
    columns = [column.id]
  }

  index "casbin_rule_idx" {
    unique  = true
    columns = [
      column.ptype, column.v0, column.v1, column.v2,
      column.v3, column.v4, column.v5
    ]
  }
}
