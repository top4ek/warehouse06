# spec/factories/nodes.rb
require 'securerandom' # For SecureRandom.hex

FactoryBot.define do
  factory :node do
    sequence(:path) { |n| "path/to/item-#{n}.txt" } # Default to file-like paths
    name { File.basename(path, ".txt") } # Adjust name if path implies type
    description { "A description for #{name}" }
    date { Date.today }

    # Default to file attributes
    digest { "sha256-#{SecureRandom.hex(4)}" }
    size { rand(100..10000) }

    trait :directory do
      sequence(:path) { |n| "path/to/directory-#{n}" } # Override path for dirs
      name { File.basename(path) }
      digest { nil }
      size { nil }
      description { "A directory description for #{name}" }
    end
  end
end
