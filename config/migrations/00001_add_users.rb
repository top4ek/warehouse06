# frozen_string_literal: true

Sequel.migration do
  up do
    create_table(:users) do
      primary_key :id

      String :name, null: true, size: 20
      String :password, null: false, size: 72
      String :email, null: false, size: 50

      DateTime :created_at
      DateTime :updated_at

      index :id, unique: true
      index :email, unique: true
    end
  end

  down do
    drop_table(:users)
  end
end
