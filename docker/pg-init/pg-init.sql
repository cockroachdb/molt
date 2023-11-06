CREATE DATABASE molt;
\c molt;

CREATE TABLE employees (
    id serial PRIMARY KEY,
    unique_id UUID,
    name VARCHAR(50),
    created_at TIMESTAMPTZ,
    updated_at DATE,
    is_hired BOOLEAN,
    salary NUMERIC(8, 2),
    bonus REAL
);

INSERT INTO employees (unique_id, name, created_at, updated_at, is_hired, salary, bonus)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000'::uuid,
    'John Doe',
    '2023-11-03 09:00:00'::timestamp,
    '2023-11-03'::date,
    true,
    5000.00,
    100.25
);
