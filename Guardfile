guard :rspec, cmd: "bundle exec rspec" do
  watch(%r{^spec/.+_spec\.rb$})
  watch(%r{^app/([^/]+)\.rb$}) do |m|
    [
      "spec/app/#{m[1]}_spec.rb",
      "spec/models/#{m[1]}_spec.rb",
    ]
  end

  watch(%r{^app/(\w+)/(.+)\.rb$}) do |m|
    [
      "spec/#{m[1]}/#{m[2]}_spec.rb",
      "spec/requests/#{m[1]}/#{m[2]}_spec.rb"
    ]
  end

  watch('config.ru') { "spec" }
  watch('config/database.rb') { "spec" }
  watch('spec/spec_helper.rb')  { "spec" }
end

guard 'puma', port: 3000, config: 'config/puma.rb' do
  watch('Gemfile.lock')
  watch(%r{^config/.*\.rb$})
  watch(%r{^config/.*\.yml$})
  watch(%r{^app/.*\.rb$})
end
