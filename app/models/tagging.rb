# frozen_string_literal: true

class Tagging < Sequel::Model
  plugin :validation_helpers

  many_to_one :tag
  many_to_one :taggable_id

  def validate
    super
    validates_presence :tag_id, :taggable_id, :taggable_type
  end
end
