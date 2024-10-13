# frozen_string_literal: true

class Tag < Sequel::Model
  plugin :validation_helpers

  one_to_many :taggings

  raise_on_save_failure

  def validate
    super
    validates_presence :name
  end
end
