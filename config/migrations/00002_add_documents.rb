# frozen_string_literal: true

Sequel.migration do
  up do
    create_table(:documents) do
      primary_key :id
      foreign_key :directory_id, :directories, null: false

      String :key, null: false, size: 64
      String :path, null: true
      String :digest, null: true, size: 64
      BigDecimal :size, null: true

      index :id, unique: true
    end
  end

  down do
    drop_table(:documents)
  end
end
