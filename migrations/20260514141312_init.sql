-- +goose Up
CREATE TABLE departments (
                             id SERIAL PRIMARY KEY,
                             name VARCHAR(200) NOT NULL,
                             parent_id INT REFERENCES departments(id) ON DELETE CASCADE,
                             created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE employees (
                           id SERIAL PRIMARY KEY,
                           department_id INT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
                           full_name VARCHAR(200) NOT NULL,
                           position VARCHAR(200) NOT NULL,
                           hired_at DATE,
                           created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE employees;
DROP TABLE departments;