-- Stores the current balance of each character wallet and each corporation
-- wallet division, replaced on every refresh. Balance history is in the
-- wallet journals' balance column.
--   owner_type   character or corporation
--   division     1 to 7 for corporations, 0 for a character's one wallet
-- Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists wallet_balances (
    owner_id bigint not null,
    owner_type text not null check (owner_type in ('character', 'corporation')),
    division int not null check (division between 0 and 7),
    balance numeric(20,4) not null,
    updated_at timestamptz not null default now(),
    primary key (owner_id, division)
);
commit;
