-- Stores each character's loyalty points per NPC corporation, one row per
-- day, so LP can be tracked over time. The day's row is replaced on every
-- refresh, so it holds the last balance seen that day. Only characters hold
-- LP; ESI has nothing for player corporations.
--   lp_corporation_id   the NPC corporation the LP are with
-- Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists loyalty_points (
    character_id bigint not null,
    lp_corporation_id bigint not null,
    date date not null,
    loyalty_points bigint not null,
    updated_at timestamptz not null default now(),
    primary key (character_id, lp_corporation_id, date)
);
commit;
