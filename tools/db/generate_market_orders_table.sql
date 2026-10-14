-- Stores every market order of each character and corporation, open and
-- past. Open orders are updated on every refresh; once an order closes, ESI's
-- order history gives its final state and the row keeps it after ESI forgets
-- it (90 days).
--   owner_type       character or corporation
--   state            open, cancelled or expired (expired includes filled)
--   is_corporation   for character orders: placed on the corporation's behalf
--   issued_by, wallet_division   corporation orders only
-- A corporation order shows up under the corporation, and also under the
-- character who placed it when that character's orders are read.
-- Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists market_orders (
    owner_id bigint not null,
    owner_type text not null check (owner_type in ('character', 'corporation')),
    order_id bigint not null,
    type_id bigint not null,
    is_buy_order bool not null,
    price numeric(20,2) not null,
    volume_total bigint not null,
    volume_remain bigint not null,
    min_volume bigint,
    escrow numeric(20,2),
    range text not null,
    duration int not null,
    issued timestamptz not null,
    location_id bigint not null,
    region_id bigint not null,
    is_corporation bool,
    issued_by bigint,
    wallet_division int,
    state text not null,
    updated_at timestamptz not null default now(),
    primary key (owner_id, order_id)
);
create index if not exists market_orders_type_id_idx on market_orders (type_id);
create index if not exists market_orders_state_idx on market_orders (owner_id, state);
commit;
