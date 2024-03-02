# protoc-gen-go-setters

will generate setter funcs for proto generated messages to simplify setting values for proto fields in particular for oneofs.

## install

```
go install github.com/lcmaguire/protoc-gen-go-setters@latest
```

## examples

```.go
// instead of setting values like below.
example := example.Example{
    Active: true,
    Foo: example.Foo{...},
    Name: "myName",
    ...
}

// or updating values like this
example.Active = true
...

// you can now
example := &example.Example{}
example.SetFoo(example.Foo{...}).SetActive(true).SetName("myName")
```

### oneofs

From
```.go
	example := example.Example{
		Sample: &example.SampleMessage{
			TestOneof: &example.SampleMessage_Name{
				Name: "abcedfg",
			},
		},
	}
```

to 
```.go
example.GetSample().SetName("abcdefg")
// or
example.Sample.SetName("abcdefg")
```

### Arrays 

generates Append functions for repeated fields 

```.go
// instead of
example.Tags = append(example.Tags, "funny", "laugh", ...)

// you can now 
example.AppendTags("funny")

// AppendX takes in ... X. so you can add as many as you want
example.AppendTags("funny", "laugh", ...)
```

you can also explicitly set the value of the array 
```.go
tags := []string{"a", "b"}
example.SetTags(tags)
```

### Map

generates functions to set values for a field 

```.go
// instead of
example.FooMap["key"] = value

// you can now
example.SetFooMapKey("key", value)
```

you can also explicitly set the map to overide its existing value.
```.go
myMap := map[key]value{...}
example.SetFooMap(myMap)
```


## Supported Setters

| fieldKind | supported          | repeated           | nested             |
| --------- | ------------------ | ------------------ | ------------------ |
| scalar    | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| message   | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| enum      | :white_check_mark: | :white_check_mark:                | :white_check_mark: |
| oneof     | :white_check_mark: | :moyai:            | :white_check_mark: |
| maps      | :white_check_mark:                | :moyai:            | :white_check_mark:                |


:moyai: indicates it is unsupported by proto feature see [proto-guide](https://protobuf.dev/programming-guides/proto3/)