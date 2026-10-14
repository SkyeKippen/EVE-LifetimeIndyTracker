-- Stores each character's EVE SSO tokens, saved when they sign in, so the
-- hourly refresh can keep reading their data after the server restarts.
-- The access and refresh tokens are AES-256-GCM encrypted by the app with
-- TOKEN_ENCRYPTION_KEY from config/.env, so they are bytea here. The app
-- refreshes the access token when expires_at has passed and saves the new
-- pair. Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists esi_tokens (
    character_id bigint primary key,
    character_name text not null,
    scopes text not null default '',
    access_token bytea not null,
    refresh_token bytea not null,
    expires_at timestamptz not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
commit;
