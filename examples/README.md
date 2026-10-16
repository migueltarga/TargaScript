# TargaScript Examples

This directory contains example TargaScript programs that demonstrate various language features and capabilities. These examples are designed to be clear, well-documented, and easy to follow.

## Core Language Features

### Built-in Functions

TargaScript provides a rich set of built-in functions for common operations:

#### Basic Functions

- `print(...args)` - Print values to standard output
- `printf(format, ...args)` - Print formatted values using format specifiers
- `format(format, ...args)` - Format values like `printf`, returning the text instead of printing it
- `type(arg)` - Return the type of a value as a string
- `range(start, end, step)` - Create an array with integers from start to end (inclusive), counting by step (1 when left out)

#### Type Conversion

- `toString(arg)` - Convert a value to a string
- `toInt(arg)` - Convert a value to an integer
- `toFloat(arg)` - Convert a value to a float

#### Math Functions

- `abs(num)` - Return the absolute value of a number
- `sqrt(num)` - Return the square root of a number
- `pow(base, exponent)` - Return base raised to the power of exponent (or use `base ** exponent`, which keeps integers as integers)
- `round(num, places)` - Round a number to the nearest integer, or to a number of decimal places
- `floor(num)` - Round a number down to the nearest integer
- `ceil(num)` - Round a number up to the nearest integer
- `min(a, b, ...)` / `min(list)` - Return the smallest number
- `max(a, b, ...)` / `max(list)` - Return the largest number
- `random()` / `random(a, b)` / `random(list)` - A float between 0 and 1, an integer from a to b, or a random element

#### JSON

- `parseJSON(text)` - Convert JSON text to values. Objects keep the order of their keys, and numbers without a fraction or exponent become integers
- `toJSON(value)` / `toJSON(value, indent)` - Convert a value to JSON text, indented by a number of spaces or a string

#### Asynchronous Functions

- `sleep(ms)` - Return a promise that resolves after `ms` milliseconds, for `await sleep(500)`
- `fetch(url, options)` - Make an HTTP request and return a promise of the response, with `status`, `ok`, `url`, `headers`, `text` and `json()`. Options are `method`, `headers` and `body`

#### Command Line

These work when running scripts from the command line, not in the browser. Paths are relative to the working directory.

- `readFile(path)` - The content of a file
- `writeFile(path, text)` / `appendFile(path, text)` - Replace the content of a file, or add to its end, creating it if needed
- `listDir(path)` - The names in a directory, sorted
- `env(name, default)` - An environment variable, or `default` (null if not given) when it isn't set
- `exit(code)` - End the program with an exit code, 0 by default
- `args` - The command-line arguments after the script

#### Time Functions

- `now()` - Return the current Unix timestamp (seconds since Jan 1, 1970)
- `timeFormat(timestamp, format)` - Format a Unix timestamp, e.g. `"YYYY-MM-DD HH:mm:ss"`
- `parseTime(text, format)` - Read a time written in a format like timeFormat's, as a Unix timestamp
- `clock()` - Milliseconds since the program started, for timing code

### Loop Constructs

TargaScript provides several loop constructs:

1. **Collection-based loops** - Iterate over arrays or objects:
   ```
   repeat item in collection {
     // Code to execute for each item
   }
   ```

   For an object, the loop goes over its keys in insertion order (`obj[key]` gives the value).

   With two variables, the loop gets the key and the value: the index and element of an array, string or range, or the key and value of an object:
   ```
   repeat i, name in names { print(i, name) }
   repeat key, value in settings { print(key, "=", value) }
   ```

2. **Condition-based loops** - Loop as long as a condition is true:
   ```
   repeat condition {
     // Code to execute while condition is true
   }
   ```

3. **Range-based loops** - Iterate over a numerical range:
   ```
   repeat i in start..end {
     // Code to execute for each number in the range (inclusive)
   }
   ```

   `start..end` builds the array `[start, ..., end]` (empty when start > end), like `range(start, end)`. In a `repeat` it is iterated without building the array.

4. **Loop Control Statements** - Control the flow of loops:
   ```
   break    // Exit the loop immediately
   continue // Skip to the next iteration
   ```

### Method-Style Dot Notation

TargaScript uses method-style dot notation for strings and arrays, which allows for more expressive and chainable operations:

#### Object Properties

- `.length` - Get the length of a string, array, or hash

#### String Methods

Strings support the following method-style operations:

- `str.length` - Get the length of the string
- `str.trim()` - Remove whitespace from both ends
- `str.lower()` - Convert to lowercase
- `str.upper()` - Convert to uppercase
- `str.replace(old, new)` - Replace all occurrences of 'old' with 'new'
- `str.split(delimiter)` - Split into an array of substrings
- `str.contains(sub)` - Whether `sub` appears in the string
- `str.startsWith(prefix)` / `str.endsWith(suffix)` - Check the start or end
- `str.indexOf(sub)` - Position of the first `sub`, or -1
- `str[i]` - The character at position `i` (`null` when out of range)
- `str.repeat(n)` - The string repeated `n` times
- `str.padStart(width, fill)` / `str.padEnd(width, fill)` - Add `fill` (a space by default) until the string is `width` characters long
- `str.reverse()` - The characters in reverse order
- `str.slice(start, end)` - The characters from `start` up to (not including) `end`; negative positions count from the end, and `end` is optional
- `str.chars()` - An array of the characters
- `str.lines()` - An array of the lines, without their line breaks
- `str.trimStart()` / `str.trimEnd()` - Remove whitespace from one end
- `str.count(sub)` - How many times `sub` appears
- `str.capitalize()` - The string with its first character in uppercase

`repeat c in str { ... }` iterates over the characters of a string.

`"${value}"` puts any value in a string, and `"${value:spec}"` formats it like `format("%spec", value)`: `"${price:.2f}"`, `"${count:5d}"`, `"${name:-10s}"`.

Example of string method chaining:
```
// Using method-style notation
print("  hello  ".trim().upper())  // Prints "HELLO"
```

#### Array Methods

Arrays support the following method-style operations:

- `array.first()` - Get the first element
- `array.last()` - Get the last element
- `array.rest()` - Get all elements except the first
- `array.insert(value)` - Append a value to the end of the array (changes the array) and return it
- `array.join(separator)` - Join into a string with the separator
- `array.map(function)` - Transform each element
- `array.filter(function)` - Filter elements by a condition
- `array.reduce(function, initialValue)` - Reduce to a single value
- `array.contains(value)` - Whether the array has an element equal to `value`
- `array.indexOf(value)` - Position of the first element equal to `value`, or -1
- `array.find(function)` - First element for which the function returns true, or `null`
- `array.slice(start, end)` - Copy of the elements from `start` up to (not including) `end`; negative positions count from the end, and `end` is optional
- `array.sort()` / `array.sort(compare)` - Sort the array in place and return it. Without `compare`, elements must be all numbers or all strings; `compare(a, b)` returns a negative number when `a` comes first
- `array.reverse()` - Reverse the array in place and return it
- `array.some(function)` / `array.every(function)` - Whether the function returns true for some or for every element
- `array.count(value)` / `array.count(function)` - How many elements equal `value`, or make the function return true
- `array.sum()` - The sum of the numbers in the array
- `array.min()` / `array.max()` - The smallest or largest element, or `null` for an empty array
- `array.unique()` - The elements without repeats, in their first order
- `array.flatten(depth)` - Nested arrays replaced by their elements, `depth` levels deep (1 by default)
- `array.chunk(size)` - The elements in arrays of `size`, the last one possibly shorter
- `array.zip(other)` - Pairs `[a, b]` of the elements at the same position, as many as the shorter array has
- `array.toObject()` - An object from `[key, value]` pairs, the opposite of `object.entries()`
- `array.groupBy(function)` - An object of arrays, grouping the elements by what the function returns
- `array.sortBy(function)` - Sort the array in place by what the function returns for each element, and return it
- `array.pop()` / `array.shift()` - Remove and return the last or first element, or `null` when empty
- `array.unshift(value)` - Add a value at the start of the array and return it
- `array.removeAt(position)` - Remove and return the element at a position; negative positions count from the end

#### Object Methods

- `object.keys()` / `object.values()` - The keys or values, in insertion order
- `object.entries()` - `[key, value]` pairs, in insertion order
- `object.has(key)` - Whether the object has a key
- `object.get(key, default)` - The value of a key, or `default` (null if not given) when it's missing
- `object.remove(key)` - Remove a key and return its value

Example of array method chaining:
```
// Using method-style notation
let result = [1, 2, 3, 4, 5]
  .filter(fn(x) { return x % 2 == 0 })
  .map(fn(x) { return x * 2 })
  .join(",")
// Result: "4,8"
```

#### Complex Transformations

Method-style notation is particularly useful for complex data transformations, making the code more readable and maintainable:

```
// Process an array of strings and join the results
let processed = ["hello", "WORLD", "  TargaScript  "]
  .map(fn(word) { return word.trim().lower() })
  .join(" ")
// Result: "hello world targascript"

// Parse and transform structured data
let entries = "NAME:John,AGE:30,CITY:New York"
  .split(",")
  .map(fn(entry) { 
    let parts = entry.split(":")
    return {
      "key": parts[0].lower(),
      "value": parts[1].trim()
    }
  })
```

For comprehensive examples, see `dot_notation.tg` which demonstrates all available methods and various chaining patterns.

## Example Files

### Fundamentals

1. **functions_demo.tg** - Comprehensive demonstration of built-in functions
   - Type conversion, type inspection, and properties
   - Math functions
   - Method-style string and array operations

2. **dot_notation.tg** - Demonstration of method-style dot notation syntax
   - String methods (trim, lower, upper, replace, split)
   - Array methods (join, map, filter, reduce)
   - Method chaining for complex transformations
   - Combining string and array methods

3. **loops.tg** - Comprehensive examples of loop constructs
   - Collection-based loops, condition-based loops, and range-based loops
   - Loop control with break and continue
   - Nested loops and practical applications

4. **logical_operators_comprehensive.tg** - Complete exploration of logical operators
   - Basic boolean operations
   - Truthy/falsy values across different types
   - Short-circuit evaluation and operator precedence

5. **format_test.tg** - Demonstrates printf and format specifiers
   - Basic format specifiers for all data types
   - Multiple values and expressions
   - Advanced formatting techniques and practical examples

6. **operators.tg** - Demonstrates basic operators
   - Arithmetic operations (+, -, *, /, %)
   - Comparison operators (==, !=, <, >, <=, >=)
   - Logical operators (&&, ||, !)
   - String concatenation
   - Increment and decrement (`x++`, `x--`)

### Data Manipulation

1. **string_operations.tg** - String concatenation, manipulation, and formatting
   - String creation and concatenation
   - Method-style string manipulation 
   - String formatting techniques

2. **array_transformations.tg** - Working with arrays and transformations
   - Array creation and manipulation
   - Using array methods (map, filter, reduce)
   - Practical array transformation examples

3. **data_processor.tg** - Practical data processing with method chaining
   - Processing text data using method chains
   - Filtering and transforming collections of objects
   - Statistical analysis with chained operations
   - Combining multiple transformations in a single chain

4. **math_string_time.tg** - Math calculations, string operations, and time functions
   - Math operations and functions
   - String manipulation techniques
   - Date and time handling

5. **word_stats.tg** - Word statistics for a short text
   - String interpolation with `${...}`, ranges with `a..b` and `x++`
   - Iterating over strings and objects
   - Sorting, slicing and searching arrays
   - Loading helpers from `lib/text.tg` with `load ... as`

### Applications and Algorithms

1. **fibonacci.tg** - Multiple implementations of the Fibonacci sequence
   - Recursive, iterative, and dynamic programming approaches
   - Performance comparisons and optimization techniques

2. **calculator.tg** - A simple calculator implementation
   - Basic arithmetic operations
   - Scientific functions
   - Formula calculations

3. **prime_numbers.tg** - Functions for working with prime numbers
   - Prime number checking
   - Prime number generation
   - Prime factorization

4. **todo_list.tg** - A todo list application demonstrating state management
   - Object-oriented programming style
   - State management patterns
   - CRUD operations (Create, Read, Update, Delete)

5. **contacts_manager.tg** - A contacts management application
   - Contact creation and storage
   - Searching and filtering contacts
   - Contact modification and deletion

6. **sorting.tg** - Implementation of sorting algorithms
   - Bubble sort implementation
   - Algorithm benchmarking

7. **function_scoping_test.tg** - Test cases for function parameter scoping
   - Demonstrates how function parameters shadow outer variables
   - Shows scope isolation between function calls
   - Tests nested function scoping behavior

8. **if_scoping_test.tg** - Test cases for block scoping in if statements
   - Demonstrates variable shadowing in if blocks
   - Shows block-level variable scope isolation
   - Tests nested block scoping behavior

9. **constants.tg** - How constant declarations behave
   - Demonstrates using the `const` keyword for immutable variables
   - Shows how constants behave with different data types
   - Tests reassignment protection for constants

## Running Examples

To run any example, use:

```
go run ./cmd/targascript examples/filename.tg
```

For example:

```
go run ./cmd/targascript examples/functions_demo.tg
```

## Creating Your Own Examples

Feel free to modify these examples or create your own! TargaScript is designed to be intuitive and expressive, making it easy to experiment with different programming concepts.

If you create an interesting example, consider contributing it back to the main repository. 