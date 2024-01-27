# frozen_string_literal: true

Sequel::Model.default_set_fields_options[:missing] = :skip
Sequel::Model.plugin :timestamps

require_relative 'models/user'
