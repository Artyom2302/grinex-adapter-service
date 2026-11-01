-- +goose Up
CREATE TABLE rates(
    id BIGSERIAL primary key,
    ask decimal(20, 8) not null ,
    bid decimal(20, 8) not null ,
    method smallint not null  ,
    n int not null ,
    m int,
    requested_at timestamptz not null
);
CREATE INDEX rqd_at_index on rates(requested_at);

-- +goose Down
drop table if exists rates;
