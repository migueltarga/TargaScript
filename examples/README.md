# TargaScript Examples

This directory contains example TargaScript programs that demonstrate various language features and capabilities. These examples are designed to be clear, well-documented, and easy to follow.

## Core Language Features

### Built-in Functions

TargaScript provides a rich set of built-in functions for common operations:

#### Basic Functions

- `print(...args)` - Print values to standard output
- `printf(format, ...args)` - Print formatted values using format specifiers
- `type(arg)` - Return the type of a value as a string
- `range(start, end)` - Create an array with integers from start to end (inclusive)

#### Type Conversion

- `toString(arg)` - Convert a value to a string
- `toInt(arg)` - Convert a value to an integer
- `toFloat(arg)` - Convert a value to a float

#### Math Functions

- `abs(num)` - Return the absolute value of a number
- `sqrt(num)` - Return the square root of a number
- `pow(base, exponent)` - Return base raised to the power of exponent
- `round(num)` - Round a number to the nearest integer
- `floor(num)` - Round a number down to the nearest integer
- `ceil(num)` - Round a number up to the nearest integer
- `min(a, b)` - Return the smaller of two numbers
- `max(a, b)` - Return the larger of two numbers
- `random()` - Generate a random number between 0 and 1

#### Time Functions

- `now()` - Return the current Unix timestamp (seconds since Jan 1, 1970)
- `timeFormat(timestamp, format)` - Format a Unix timestamp using the provided format string

### Loop Constructs

TargaScript provides several loop constructs:

1. **Collection-based loops** - Iterate over arrays or objects:
   ```
   repeat item in collection {
     // Code to execute for each item
   }
   ```

2. **Condition-based loops** - Loop as long as a condition is true:
   ```
   repeat condition {
     // Code to execute while condition is true
   }
   ```

3. **Range-based loops** - Iterate over a numerical range:
   ```
   repeat i in range(start, end) {
     // Code to execute for each number in the range
   }
   ```

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
- `array.insert(value)` - Insert a value at the end
- `array.join(separator)` - Join into a string with the separator
- `array.map(function)` - Transform each element
- `array.filter(function)` - Filter elements by a condition
- `array.reduce(function, initialValue)` - Reduce to a single value

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

9. **const_test.tg** - Test cases for constant variable declarations
   - Demonstrates using the `const` keyword for immutable variables
   - Shows how constants behave with different data types
   - Tests reassignment protection for constants

## Running Examples

To run any example, use:

```
go run main.go examples/filename.tg
```

For example:

```
go run main.go examples/functions_demo.tg
```

## Creating Your Own Examples

Feel free to modify these examples or create your own! TargaScript is designed to be intuitive and expressive, making it easy to experiment with different programming concepts.

If you create an interesting example, consider contributing it back to the main repository. 