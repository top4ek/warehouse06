# frozen_string_literal: true

class User < Sequel::Model
  plugin :validation_helpers

  raise_on_save_failure

  def self.to_link
    '/users'
  end

  def to_link
    "#{self.class.to_link}/#{id}"
  end

  def validate
    super
    validates_type String, %i[name email password], allow_missing: true
    validates_presence %i[email password]
    validates_unique :email
    validates_min_length 5, :email, allow_missing: true
    validates_max_length 50, :email, allow_missing: true
    validates_max_length 20, :name, allow_missing: true
  end
end
