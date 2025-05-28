# Guardfile
# More info at https://github.com/guard/guard#readme

# RSpec plugin
# More info at https://github.com/guard/guard-rspec#readme
guard :rspec, cmd: "bundle exec rspec" do
  require "guard/rspec/dsl"
  dsl = Guard::RSpec::Dsl.new(self)

  # Feel free to open issues Guard::RSpec strictly related to Guard::RSpec.
  # For RSpec issues, you should use RSpec
  # (https://github.com/rspec/rspec).

  # Watch all specs (i.e. run all specs after file save)
  watch(dsl.specs)

  # Ruby files
  watch(dsl.app_files) { |m| dsl.run_matching_specs(m) }

  # Rails specific files
  # dsl.watch_rails_files
  # dsl.watch_spec_files_for_rails_files

  # Capybara features specs
  # watch(dsl.capybara_feature_specs)

  # Capybara integration specs
  # watch(dsl.capybara_integration_specs)
  # watch(dsl.capybara_integration_spec_files)

  # Turnip features and steps
  # watch(dsl.turnip_features)
  # watch(dsl.turnip_steps)
end

# You can also add other Guard plugins here, e.g. for RuboCop, livereload, etc.
# For example:
# guard :rubocop do
#   watch(/.+\.rb$/)
#   watch(/.+\.rake$/)
#   watch(%r{(?:Guardfile|Gemfile|Rakefile|\.ru|\.gemspec)$})
# end
