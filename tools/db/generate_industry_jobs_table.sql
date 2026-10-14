-- Stores every industry job of each character and corporation. Jobs are
-- updated on every refresh until they finish; ESI forgets finished jobs after
-- 90 days but the rows stay.
--   owner_type    character or corporation
--   activity_id   1 manufacturing, 3 TE research, 4 ME research, 5 copying,
--                 8 invention, 9 and 11 reactions
--   status        active, cancelled, delivered, paused, ready or reverted
--   location_id   the station or structure the job runs in
-- Optional dates and the completing character are NULL until they apply.
-- Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists industry_jobs (
    owner_id bigint not null,
    owner_type text not null check (owner_type in ('character', 'corporation')),
    job_id bigint not null,
    activity_id int not null,
    status text not null,
    installer_id bigint not null,
    blueprint_id bigint not null,
    blueprint_type_id bigint not null,
    blueprint_location_id bigint not null,
    output_location_id bigint not null,
    facility_id bigint not null,
    location_id bigint not null,
    product_type_id bigint,
    runs int not null,
    licensed_runs int,
    successful_runs int,
    probability double precision,
    cost numeric(20,2),
    duration int not null,
    start_date timestamptz not null,
    end_date timestamptz not null,
    pause_date timestamptz,
    completed_date timestamptz,
    completed_character_id bigint,
    updated_at timestamptz not null default now(),
    primary key (owner_id, job_id)
);
create index if not exists industry_jobs_status_idx on industry_jobs (owner_id, status);
create index if not exists industry_jobs_product_idx on industry_jobs (product_type_id);
commit;
