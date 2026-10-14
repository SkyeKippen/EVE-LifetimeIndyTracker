-- Stores what was mined at each corporation mining observer (a moon drill
-- refinery), from ESI's GET /corporation/{corporation_id}/mining/observers/
-- {observer_id}. ESI only keeps 30 days, so the app saves it every hour and
-- this table keeps everything. One row per observer, day, miner and ore type;
-- the app replaces the row each time since ESI updates the day's quantity.
--   observer_name, solar_system_*   the structure and where it is, when the
--                                   refreshing character can see it
--   recorded_corporation_id         the miner's corporation at the time
--   ore_type, base_ore              as in mining_ledger
-- Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists corp_mining_ledger (
    corporation_id bigint not null,
    observer_id bigint not null,
    observer_name text not null default '',
    solar_system_id bigint,
    solar_system_name text not null default '',
    date date not null,
    character_id bigint not null,
    character_name text not null default '',
    recorded_corporation_id bigint not null,
    type_id bigint not null,
    ore_type text not null,
    base_ore text not null,
    quantity bigint not null,
    updated_at timestamptz not null default now(),
    primary key (corporation_id, observer_id, date, character_id, type_id)
);
create index if not exists corp_mining_ledger_base_ore_idx on corp_mining_ledger (corporation_id, base_ore);
create index if not exists corp_mining_ledger_character_idx on corp_mining_ledger (character_id);
commit;
