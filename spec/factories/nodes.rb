# frozen_string_literal: true

require 'securerandom'

FactoryBot.define do
  factory :node do
    sequence(:path) { |n| "path/to/item-#{n}.txt" }
    name { File.basename(path, ".txt") }
    description { "A description for #{name}" }
    date { Date.today }

    digest { "sha256-#{SecureRandom.hex(4)}" }
    size { rand(100..10000) }

    trait :directory do
      sequence(:path) { |n| "path/to/directory-#{n}" }
      name { File.basename(path) }
      digest { nil }
      size { nil }
      description { "A directory description for #{name}" }
    end
  end
end
