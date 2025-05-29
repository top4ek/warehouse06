# frozen_string_literal: true

require 'securerandom' # For SecureRandom.hex

FactoryBot.define do
  factory :node do
    sequence(:name) { |n| "Node_#{n}" }
    sequence(:path) { |n| "path/node_#{n}" } # Ensure path is unique
    description { "Default directory description." } # For directory type
    digest { nil } # Default to directory type

    trait :file do
      sequence(:name) { |n| "file_#{n}.txt" }
      sequence(:path) { |n| "files/file_#{n}.txt" } # Ensure path is unique
      description { nil }
      digest { SecureRandom.hex(32) } # Ensures type is 'file'
    end

    trait :directory do
      # This trait is mostly for clarity, as the default is directory-like
      sequence(:name) { |n| "dir_#{n}" }
      sequence(:path) { |n| "dirs/dir_#{n}" } # Ensure path is unique
      description { "## Directory Content for #{name}" }
      digest { nil }
    end
  end
end
