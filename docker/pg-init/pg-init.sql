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

DO $$ 
DECLARE 
    i INT;
BEGIN
    i := 1;
    WHILE i <= 200000 LOOP
        INSERT INTO employees (unique_id, name, created_at, updated_at, is_hired, salary, bonus)
        VALUES (
            ('550e8400-e29b-41d4-a716-446655440000'::uuid),
            'Employee_' || i,
            '2023-11-03 09:00:00'::timestamp,
            '2023-11-03'::date,
            true,
            5000.00,
            100.25
        );
        i := i + 1;
    END LOOP;
END $$;
