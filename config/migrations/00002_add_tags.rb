# frozen_string_literal: true

Sequel.migration do
  up do
    create_table(:tags) do
      primary_key :id

      String :key, null: false, size: 64

      index :id, unique: true
    end

    create_table :taggings do
      primary_key :id
      foreign_key :tag_id, :tags, null: false

      BigDecimal :taggable_id, null: false
      String :taggable_type, null: false, size: 32

      index %i[tag_id taggable_id taggable_type]
    end
  end

  down do
    drop_table(:tags)
    drop_table(:taggings)
  end
end
