-- Stores every delivered manufacturing and reaction job once, with how many
-- items it made, so lifetime totals per item can be added up long after ESI
-- forgets the job (it keeps finished jobs for 90 days). The app fills it
-- from industry_jobs after every refresh, so jobs saved before this table
-- existed are counted too. A job seen in both a character's and their
-- corporation's job list is stored once.
--   activity_id     1 manufacturing, 9 or 11 reactions
--   installer_id    the character who started the job, whoever owned it
--   units_per_run   how many items one run makes, from the SDE (e.g. 100
--                   for most ammo)
--   quantity        runs x units_per_run, the items delivered
--   unit_price      the item's Jita sell price on the day the job was
--                   delivered, or the closest day with a saved price; NULL
--                   until a price is found
--   value           quantity x unit_price
-- Run after generate_industry_jobs_table.sql. Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists delivered_jobs (
    job_id bigint primary key,
    activity_id int not null,
    installer_id bigint not null,
    installer_name text not null default '',
    blueprint_type_id bigint not null,
    product_type_id bigint not null,
    product_name text not null default '',
    runs int not null,
    units_per_run int not null,
    quantity bigint not null,
    facility_id bigint not null,
    completed_date timestamptz not null,
    recorded_at timestamptz not null default now()
);
-- value columns, added after the table was first created
alter table delivered_jobs add column if not exists unit_price numeric(20,2);
alter table delivered_jobs add column if not exists value numeric(24,2);
create index if not exists delivered_jobs_installer_idx on delivered_jobs (installer_id, product_type_id);
commit;
