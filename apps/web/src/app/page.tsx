import Link from 'next/link';

export default function Home() {
  return (
    <main className="min-h-screen bg-white">
      <div className="max-w-4xl mx-auto px-4 py-16">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">
          Blackbox-AgentDiff
        </h1>
        <p className="text-xl text-gray-600 mb-8">
          Flight Recorder for AI Agents — Open-source observability with execution diffing.
        </p>

        <div className="bg-gray-50 border border-gray-200 rounded-lg p-6 mb-8">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Quick Links</h2>
          <div className="flex gap-4">
            <Link
              href="/dashboard"
              className="px-4 py-2 bg-black text-white rounded hover:bg-gray-800"
            >
              Dashboard
            </Link>
            <Link
              href="/traces"
              className="px-4 py-2 bg-gray-200 text-gray-900 rounded hover:bg-gray-300"
            >
              View Traces
            </Link>
            <Link
              href="/metrics"
              className="px-4 py-2 bg-gray-200 text-gray-900 rounded hover:bg-gray-300"
            >
              Metrics
            </Link>
            <Link
              href="/settings"
              className="px-4 py-2 bg-gray-200 text-gray-900 rounded hover:bg-gray-300"
            >
              Settings
            </Link>
          </div>
        </div>

        <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-blue-900 mb-2">Get Started</h2>
          <p className="text-blue-800 mb-4">
            Configure your API connection in Settings, then send traces via the Python or TypeScript SDK.
          </p>
          <pre className="bg-blue-900 text-blue-100 p-4 rounded text-sm overflow-x-auto">
{`# Python
import blackbox
blackbox.init()
with blackbox.trace("my-agent") as trace:
    ...

# TypeScript
import { Blackbox } from 'blackbox-agentdiff'
const bb = new Blackbox()
const trace = bb.trace('my-agent')`}
          </pre>
        </div>
      </div>
    </main>
  );
}