# Todo

1. logging
   1. logging needs to be standardized for the proxy.
   2. logging needs to be standardized for the example server, as it will eventually be used in e2e tests
2. tests
   1. need to come up with better mocks and extract common logic to help tests appear smaller and be easier to read and understand
   2. when you do the above, make sure to be able to return actual errors
3. error handling
   1. error handling is not standardized in any way and there is quite a bit of cyclomatic complexity related to it. it needs to be cleaned up
4. managing goroutines
   1. is there a way to reliably test that all goroutines have been cleaned up after work is done by the proxy?