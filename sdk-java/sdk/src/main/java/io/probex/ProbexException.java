package io.probex;

/** Exception thrown by the PROBEX SDK when an API call fails. */
public class ProbexException extends Exception {

    public ProbexException(String message) {
        super(message);
    }

    public ProbexException(String message, Throwable cause) {
        super(message, cause);
    }
}
