# frozen_string_literal: true

Sequel.migration do
  up do
    # , using: 'fts5(description, name, key
    create_table(:directories) do
      primary_key :id
      foreign_key :parent_id, :directories

      String :key, null: false, size: 64
      String :name, null: true, size: 64
      String :description, null: true, text: true

      Date :date, null: true
    end
  end

  down do
    drop_table(:directories)
  end
end
