/**
 * Cloudflare Worker for ERP AI Edge Gateway
 * Runs LLM inference directly on Cloudflare Edge GPUs using Workers AI.
 *
 * Deploy with:
 *   npx wrangler deploy
 */

export default {
  async fetch(request, env, ctx) {
    // 1. Handle CORS Preflight
    if (request.method === "OPTIONS") {
      return new Response(null, {
        headers: {
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
          "Access-Control-Allow-Headers": "Content-Type, Authorization",
        },
      });
    }

    const url = new URL(request.url);

    // Health check endpoint
    if (url.pathname === "/health" || url.pathname === "/") {
      return new Response(JSON.stringify({
        status: "ok",
        service: "GoERP Cloudflare Workers AI Gateway",
        models: {
          default_llm: "@cf/meta/llama-3.1-8b-instruct",
          deep_rca: "@cf/meta/llama-3.3-70b-instruct",
          vision_ocr: "@cf/meta/llama-3.2-11b-vision-instruct"
        }
      }), {
        headers: { "Content-Type": "application/json", "Access-Control-Allow-Origin": "*" }
      });
    }

    // AI Inference Endpoint
    if (request.method === "POST") {
      try {
        const body = await request.json();
        const model = body.model || "@cf/meta/llama-3.1-8b-instruct";
        const messages = body.messages || [
          { role: "system", content: "You are an intelligent ERP Enterprise Copilot." },
          { role: "user", content: body.prompt || "Hello" }
        ];

        // Call Cloudflare Workers AI Binding
        const aiResponse = await env.AI.run(model, {
          messages: messages,
          max_tokens: body.max_tokens || 1024,
          temperature: body.temperature || 0.2
        });

        return new Response(JSON.stringify({
          success: true,
          model: model,
          response: aiResponse.response || aiResponse
        }), {
          headers: {
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*"
          }
        });
      } catch (err) {
        return new Response(JSON.stringify({
          success: false,
          error: err.message
        }), {
          status: 500,
          headers: {
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*"
          }
        });
      }
    }

    return new Response("Not Found", { status: 404 });
  }
};
