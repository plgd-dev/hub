# Developer Guidelines

## Golang code

### Formatting

Format your code using `go fmt -s`. Formatting of pushed code is automatically checked by GitHub action.

### Style

General guideline is to follow the code conventions of the surrounding code to keep the style within files consistent.

The following rules are recommendations and are not required to be _always_ followed. Just use common sense to discern when to ignore a recommendation.

#### YAML tags

Services are configured using `.yaml` files, which are unmarshaled into a Golang structures on service start. Therefore there must be a strict correspondence between keys in `.yaml` files and YAML tags on the field of Golang configuration structure.

For field names and YAML tags `camelCase` is used. Abbreviations and acronyms are all upper-case; except for tags at the beginning of a word when they should be all lower-case. Tag name should start with a lower-case letter, but otherwise should copy the field name.

Example:

```Go
type Example struct {
  Event     string        `yaml:"event"`
  EventURL  string        `yaml:"eventURL"`
  ExpiresIn time.Duration `yaml:"expiresIn"`
  ID        string        `yaml:"id"`
}
```

#### JSON tags

Marshaling to and unmarshaling from JSON is used by structures that are part of public API. Therefore there might be a specification that defines how the marshaled fields must be named in a payload (for example: `subscriptionId`, `eventsUrl` from [OCF Cloud API for Cloud Services Specification](https://openconnectivity.org/specs/OCF_Cloud_API_For_Cloud_Services_Specification_v2.2.4.pdf)).

For JSON tags strict `camelCase` is used, abbreviations and acronyms have the first letter upper-case and the rest is lower-case. Tag name should start with a lower-case letter and should usually copy the field name (this rule might be broken when a field is required to have a specific name).

Example:

```Go
type Example struct {
  Event     string        `json:"event"`
  EventURL  string        `yaml:"eventUrl"`
  ExpiresIn time.Duration `yaml:"expiresIn"`
  ID        string        `yaml:"id"`
}
```
