CREATE TABLE capital_gains_balance (
                                       account_id INT NOT NULL,
                                       tax_year INT NOT NULL,
                                       net_capital_change  BIGINT NOT NULL,
                                       carryover_loss INT NOT NULL,
                                       primary key (account_id, tax_year)

);


CREATE TABLE transaction_history (
                                     account_id INT NOT NULL,
                                     order_id INT NOT NULL,
                                     activity_id INT NOT NULL,
                                     stock_ticker VARCHAR(10) NOT NULL,
                                     share_count INT NOT NULL ,
                                     stock_price DECIMAL(19,4) NOT NULL ,
                                     order_type VARCHAR(4) CHECK (order_type IN ('BUY', 'SELL')) NOT NULL,
                                     is_completed BOOLEAN DEFAULT FALSE,
                                     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                     primary key (account_id, order_id, activity_id)
);

ALTER TABLE transaction_history
ADD COLUMN matched BOOLEAN NOT NULL default false;

create function upsertcapitalchangebalance(p_account_id integer, p_tax_year integer, p_net_capital_change bigint, p_carryover_loss integer) returns void
    language plpgsql
as
$$
BEGIN
    -- Try to update the record
    UPDATE capital_gains_balance
    SET net_capital_change = net_capital_change + p_net_capital_change
    WHERE account_id = p_account_id AND tax_year = p_tax_year;

    -- If no row was updated, insert a new record
    IF NOT FOUND THEN
        INSERT INTO capital_gains_balance (account_id, tax_year, net_capital_change, carryover_loss)
        VALUES (p_account_id, p_tax_year, p_net_capital_change, p_carryover_loss);
    END IF;
END;
$$;

alter function upsertcapitalchangebalance(integer, integer, bigint, integer) owner to postgres;
