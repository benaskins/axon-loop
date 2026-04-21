// Package loop provides a provider-agnostic conversation loop for
// LLM-powered agents. It handles message exchange, tool call dispatch,
// and streaming — with no HTTP, persistence, or UI concerns.
//
// Class: primitive
// UseWhen: Any LLM interaction in libraries or HTTP services. For CLI agents, use axon-hand instead which provides axon-loop automatically. Always paired with axon-talk and axon-tool.
package loop
