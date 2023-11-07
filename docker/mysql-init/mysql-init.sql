CREATE DATABASE molt;
use molt;
CREATE TABLE employees (
    id INT AUTO_INCREMENT PRIMARY KEY,
    unique_id BINARY(16),
    name VARCHAR(50),
    created_at DATETIME,
    updated_at DATE,
    is_hired TINYINT(1),
    salary DECIMAL(8, 2),
    bonus FLOAT
);

DELIMITER $$
CREATE PROCEDURE InsertEmployeesWithTransaction()
BEGIN
    DECLARE i INT;
    SET i = 1;
    
    START TRANSACTION;

    WHILE i <= 200000 DO
        INSERT INTO employees (unique_id, name, created_at, updated_at, is_hired, salary, bonus)
        VALUES (
            UNHEX(REPLACE('550e8400-e29b-41d4-a716-446655440000', '-', '')),
            CONCAT('Employee_', i),
            '2023-11-03 09:00:00',
            '2023-11-03',
            1,
            5000.00,
            100.25
        );
        SET i = i + 1;
    END WHILE;

    COMMIT;
END$$
DELIMITER ;

CALL InsertEmployeesWithTransaction();

