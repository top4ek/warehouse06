# frozen_string_literal: true

module Serializers
  class User < Blueprinter::Base
    identifier :id
    fields :name, :email, :created_at
  end
end
