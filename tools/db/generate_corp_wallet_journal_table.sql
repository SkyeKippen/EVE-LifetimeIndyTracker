-- Stores corporation wallet journal entries from ESI
-- (GET /corporations/{corporation_id}/wallets/{division}/journal), one row
-- per entry per wallet division.
--   corporation_id, division  whose wallet the entry is in; ESI doesn't put
--                             these in the entry, they come from the request
--   journal_id                ESI's "id"; a transfer between two divisions of
--                             the same corporation shows in both, so the key
--                             includes the division
--   amount, balance, tax      ISK, kept exact to the 4 decimals ESI can send
--   context_id(_type)         what the entry refers to, e.g. a
--                             market_transaction_id or an industry_job_id
--   ref_type                  the kind of entry, e.g. market_escrow
-- ESI leaves out the optional fields when they don't apply, so those are
-- nullable. Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists corp_wallet_journal (
    corporation_id bigint not null,
    division int not null check (division between 1 and 7),
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
    primary key (corporation_id, division, journal_id)
);
create index if not exists corp_wallet_journal_date_idx on corp_wallet_journal (corporation_id, date);
create index if not exists corp_wallet_journal_ref_type_idx on corp_wallet_journal (corporation_id, ref_type);
commit;
