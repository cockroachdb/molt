CREATE TABLE employees (
	id SERIAL4 PRIMARY KEY,
	unique_id UUID,
	name VARCHAR(50),
	created_at TIMESTAMPTZ,
	updated_at DATE,
	is_hired BOOL,
	salary DECIMAL(8,2),
	bonus FLOAT4
);
