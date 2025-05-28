# frozen_string_literal: true

Sequel.migration do
  up do
    # , using: 'fts5(description, name, key
    create_table(:nodes) do
      primary_key :id

      String :path, null: true
      String :name, null: true, size: 64
      String :description, null: true, text: true
      BigDecimal :size, null: true
      String :digest, null: true, size: 64
      Date :date, null: true
      BigDecimal :parent_id, null: true

      index :id, unique: true
      index :path, unique: true
    end
  end

  down do
    drop_table(:nodes)
  end
end
