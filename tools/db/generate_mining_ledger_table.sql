-- Stores what each character mined, from ESI's character mining ledger
-- (GET /characters/{character_id}/mining). ESI only keeps 30 days, so the
-- app saves the ledger every hour and this table becomes the lifetime record.
-- One row per character, day, solar system and ore type; ESI updates a day's
-- quantity until the day ends, so the app replaces the row each time.
--   ore_type    the exact type mined, e.g. Veldspar II-Grade
--   base_ore    the ore it counts as, with grades and compression dropped,
--               e.g. Veldspar; what the Mining page groups by
--   quantity    units mined
-- Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists mining_ledger (
    character_id bigint not null,
    character_name text not null,
    date date not null,
    solar_system_id bigint not null,
    solar_system_name text not null default '',
    type_id bigint not null,
    ore_type text not null,
    base_ore text not null,
    quantity bigint not null,
    updated_at timestamptz not null default now(),
    primary key (character_id, date, solar_system_id, type_id)
);
create index if not exists mining_ledger_base_ore_idx on mining_ledger (base_ore);
commit;
