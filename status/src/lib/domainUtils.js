/**
 * Format a name into domain-friendly format (lowercase with hyphens)
 * @param {string} name - Name to format
 * @returns {string} Formatted domain-friendly name
 */
export const formatDomainName = (name) => {
  return name.toLowerCase().replace(/\s+/g, '-');
};

/**
 * Format a full domain with latency.space suffix
 * @param {string} name - Name to format
 * @returns {string} Full domain name
 */
export const formatFullDomain = (name) => {
  return `${formatDomainName(name)}.latency.space`;
};

/**
 * Format a moon domain with its parent planet
 * @param {string} moonName - Moon name
 * @param {string} planetName - Parent planet name
 * @returns {string} Formatted moon domain
 */
export const formatMoonDomain = (moonName, planetName) => {
  return `${formatDomainName(moonName)}.${formatDomainName(planetName)}.latency.space`;
};

/**
 * Format a target domain with a celestial body
 * @param {string} targetDomain - Target domain name
 * @param {string} celestialName - Celestial body name
 * @returns {string} Formatted target domain
 */
export const formatTargetDomain = (targetDomain, celestialName) => {
  return `${targetDomain}.${formatDomainName(celestialName)}.latency.space`;
};