-- Stores the Jita sell price of each refined material (minerals, moon
-- materials, ice products) once a day, so mined ore can be valued at the
-- prices of the day it was mined. The price is the lowest sell order at
-- Jita IV - Moon 4 - Caldari Navy Assembly Plant, from ESI's public region
-- orders. ESI has no sell price history, so prices start the day the app
-- first saves them; days before that use the closest day saved.
--   price_date   the UTC day the price was seen; the hourly refresh replaces
--                the day's row, so it holds the last price seen that day
-- Safe to run more than once.
begin;
create schema if not exists lifetime_indy;
set search_path to lifetime_indy;

create table if not exists jita_sell_prices (
    type_id bigint not null,
    price_date date not null,
    sell_price numeric(20,2) not null,
    updated_at timestamptz not null default now(),
    primary key (type_id, price_date)
);
commit;
