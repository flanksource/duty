table "oidc_public_keys" {
  schema = schema.public
  column "id" {
    null = false
    type = text
  }
  column "algorithm" {
    null    = false
    type    = text
    default = "RS256"
  }
  column "public_key" {
    null    = false
    type    = bytea
    comment = "PEM-encoded RSA public key"
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "expires_at" {
    null    = true
    type    = timestamptz
    comment = "Null means the key does not expire; set during key rotation overlap"
  }
  primary_key {
    columns = [column.id]
  }
}

table "oidc_auth_requests" {
  schema = schema.public
  column "id" {
    null = false
    type = text
  }
  column "client_id" {
    null = false
    type = text
  }
  column "redirect_uri" {
    null = false
    type = text
  }
  column "scopes" {
    null    = false
    type    = sql("text[]")
    default = sql("'{}'::text[]")
  }
  column "state" {
    null = true
    type = text
  }
  column "nonce" {
    null = true
    type = text
  }
  column "response_type" {
    null = false
    type = text
  }
  column "code_challenge" {
    null = true
    type = text
  }
  column "code_challenge_method" {
    null = true
    type = text
  }
  column "subject" {
    null    = true
    type    = text
    comment = "Set after the user completes login (person.id)"
  }
  column "auth_time" {
    null    = true
    type    = timestamptz
    comment = "When the user authenticated"
  }
  column "code" {
    null    = true
    type    = text
    comment = "Authorization code, set after login"
  }
  column "done" {
    null    = false
    type    = boolean
    default = false
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "expires_at" {
    null    = false
    type    = timestamptz
    comment = "10-minute TTL; expired rows cleaned up by background job"
  }
  primary_key {
    columns = [column.id]
  }
  index "oidc_auth_requests_code_idx" {
    columns = [column.code]
    where   = "code IS NOT NULL"
  }
  index "oidc_auth_requests_expires_at_idx" {
    columns = [column.expires_at]
  }
}

table "oidc_refresh_tokens" {
  schema = schema.public
  column "id" {
    null = false
    type = text
  }
  column "token" {
    null    = false
    type    = text
    comment = "Opaque refresh token value"
  }
  column "client_id" {
    null = false
    type = text
  }
  column "subject" {
    null    = false
    type    = text
    comment = "person.id of the authenticated user"
  }
  column "scopes" {
    null    = false
    type    = sql("text[]")
    default = sql("'{}'::text[]")
  }
  column "auth_time" {
    null    = false
    type    = timestamptz
    comment = "Time of original authentication"
  }
  column "rotation_id" {
    null    = false
    type    = text
    comment = "Groups tokens in a rotation family for replay detection"
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "expires_at" {
    null    = false
    type    = timestamptz
    comment = "30-day expiry; rotated on each use"
  }
  primary_key {
    columns = [column.id]
  }
  index "oidc_refresh_tokens_token_idx" {
    unique  = true
    columns = [column.token]
  }
  index "oidc_refresh_tokens_subject_idx" {
    columns = [column.subject]
  }
  index "oidc_refresh_tokens_rotation_id_idx" {
    columns = [column.rotation_id]
  }
  index "oidc_refresh_tokens_expires_at_idx" {
    columns = [column.expires_at]
  }
}
