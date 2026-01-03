/**
 * Simple in-memory rate limiter
 * Note: Resets on deploy/restart. For production scale, use Redis.
 */

interface RateLimitRecord {
  count: number;
  resetTime: number;
}

const store = new Map<string, RateLimitRecord>();

// Clean up expired entries every 5 minutes
setInterval(() => {
  const now = Date.now();
  for (const [key, record] of store.entries()) {
    if (now > record.resetTime) {
      store.delete(key);
    }
  }
}, 5 * 60 * 1000);

export interface RateLimitResult {
  success: boolean;
  limit: number;
  remaining: number;
  reset: number;
}

/**
 * Check rate limit for a given key
 * @param key - Unique identifier (e.g., API key hash or IP)
 * @param limit - Max requests per window (default: 100)
 * @param windowMs - Time window in ms (default: 60000 = 1 minute)
 */
export function rateLimit(
  key: string,
  limit = 100,
  windowMs = 60000
): RateLimitResult {
  const now = Date.now();
  const record = store.get(key);

  // New or expired window
  if (!record || now > record.resetTime) {
    store.set(key, { count: 1, resetTime: now + windowMs });
    return {
      success: true,
      limit,
      remaining: limit - 1,
      reset: now + windowMs,
    };
  }

  // Within window
  if (record.count >= limit) {
    return {
      success: false,
      limit,
      remaining: 0,
      reset: record.resetTime,
    };
  }

  record.count++;
  return {
    success: true,
    limit,
    remaining: limit - record.count,
    reset: record.resetTime,
  };
}

/**
 * Create rate limit response with proper headers
 */
export function rateLimitResponse(result: RateLimitResult): Response {
  return new Response(
    JSON.stringify({ error: 'Too many requests' }),
    {
      status: 429,
      headers: {
        'Content-Type': 'application/json',
        'X-RateLimit-Limit': String(result.limit),
        'X-RateLimit-Remaining': '0',
        'X-RateLimit-Reset': String(Math.ceil(result.reset / 1000)),
        'Retry-After': String(Math.ceil((result.reset - Date.now()) / 1000)),
      },
    }
  );
}
