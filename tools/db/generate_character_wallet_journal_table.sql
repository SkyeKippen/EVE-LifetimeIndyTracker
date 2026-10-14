-- Stores character wallet journal entries from ESI
-- (GET /characters/{character_id}/wallet/journal). ESI only keeps 30 days,
-- so the app saves the journal every hour and this table keeps everything.
-- Same columns as corp_wallet_journal, without the division.
-- Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists character_wallet_journal (
    character_id bigint not null,
    journal_id bigint not null,
    date timestamptz not null,
    ref_type text not null,
    description text not null,
    amount numeric(20,4),
    balance numeric(20,4),
    context_id bigint,
    context_id_type text,
    first_party_id bigint,
    second_party_id bigint,
    reason text,
    tax numeric(20,4),
    tax_receiver_id bigint,
    fetched_at timestamptz not null default now(),
    primary key (character_id, journal_id)
);
create index if not exists character_wallet_journal_date_idx on character_wallet_journal (character_id, date);
create index if not exists character_wallet_journal_ref_type_idx on character_wallet_journal (character_id, ref_type);
commit;
