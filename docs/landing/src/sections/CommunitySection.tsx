import { useRef, useLayoutEffect, useState, useEffect } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { Github, MessageCircle, Star, GitFork, Tag } from 'lucide-react';

gsap.registerPlugin(ScrollTrigger);

interface CommunitySectionProps {
  className?: string;
}

interface GitHubStats {
  stars: number;
  forks: number;
  releases: number;
}

const REPO_OWNER = 'get-virgil';
const REPO_NAME = 'cracker-barrel';

export default function CommunitySection({ className = '' }: CommunitySectionProps) {
  const sectionRef = useRef<HTMLDivElement>(null);
  const headerRef = useRef<HTMLDivElement>(null);
  const statsRef = useRef<(HTMLDivElement | null)[]>([]);
  const ctaRef = useRef<HTMLDivElement>(null);

  const [githubStats, setGithubStats] = useState<GitHubStats>({
    stars: 0,
    forks: 0,
    releases: 0,
  });

  // Fetch GitHub stats
  useEffect(() => {
    const fetchGitHubStats = async () => {
      try {
        // Fetch repo info
        const repoResponse = await fetch(`https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}`);
        const repoData = await repoResponse.json();

        // Fetch releases count
        const releasesResponse = await fetch(`https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases`);
        const releasesData = await releasesResponse.json();

        setGithubStats({
          stars: repoData.stargazers_count || 0,
          forks: repoData.forks_count || 0,
          releases: Array.isArray(releasesData) ? releasesData.length : 0,
        });
      } catch (error) {
        console.error('Failed to fetch GitHub stats:', error);
        // Keep default values (0) on error
      }
    };

    fetchGitHubStats();
  }, []);

  const stats = [
    { value: githubStats.stars.toLocaleString(), label: 'Stars', icon: Star },
    { value: githubStats.forks.toLocaleString(), label: 'Forks', icon: GitFork },
    { value: githubStats.releases.toLocaleString(), label: 'Releases', icon: Tag },
  ];

  useLayoutEffect(() => {
    const section = sectionRef.current;
    if (!section) return;

    const ctx = gsap.context(() => {
      // Header animation
      gsap.fromTo(headerRef.current,
        { y: 30, opacity: 0 },
        {
          y: 0,
          opacity: 1,
          duration: 0.6,
          ease: 'power2.out',
          scrollTrigger: {
            trigger: headerRef.current,
            start: 'top 80%',
            end: 'top 50%',
            scrub: true,
          }
        }
      );

      // Stats animation (staggered)
      statsRef.current.forEach((stat, i) => {
        if (!stat) return;
        gsap.fromTo(stat,
          { y: 40, opacity: 0 },
          {
            y: 0,
            opacity: 1,
            duration: 0.5,
            delay: i * 0.08,
            ease: 'power2.out',
            scrollTrigger: {
              trigger: stat,
              start: 'top 80%',
              end: 'top 50%',
              scrub: true,
            }
          }
        );
      });

      // CTA animation
      gsap.fromTo(ctaRef.current,
        { scale: 0.98, opacity: 0 },
        {
          scale: 1,
          opacity: 1,
          duration: 0.5,
          ease: 'power2.out',
          scrollTrigger: {
            trigger: ctaRef.current,
            start: 'top 80%',
            end: 'top 60%',
            scrub: true,
          }
        }
      );

    }, section);

    return () => ctx.revert();
  }, []);

  return (
    <section 
      ref={sectionRef}
      className={`relative w-full min-h-screen py-20 sm:py-28 ${className}`}
    >
      {/* Background Image */}
      <div className="absolute inset-0 w-full h-full">
        <img 
          src="./community_booth_bg.jpg" 
          alt="Diner Booth"
          className="w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-[#07070A]/85" />
      </div>

      {/* Content */}
      <div className="relative z-10 max-w-[920px] mx-auto px-6">
        {/* Header */}
        <div ref={headerRef} className="text-center mb-12 sm:mb-16">
          <span className="text-[#B9FF2C] text-sm font-semibold tracking-wider uppercase mb-3 block">
            Community
          </span>
          <h2 className="text-[clamp(32px,5vw,48px)] font-black text-[#F4F6FA] mb-4">
            Join the late-night crew
          </h2>
          <p className="text-[#A7ACB8] text-base sm:text-lg max-w-xl mx-auto">
            Request kernel versions, report issues, and help us keep the grill hot.
          </p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 sm:gap-6 mb-12 sm:mb-16">
          {stats.map((stat, index) => (
            <div
              key={stat.label}
              ref={el => { statsRef.current[index] = el; }}
              className="group relative bg-[#141419] border border-white/[0.08] rounded-[18px] p-6 sm:p-8 text-center hover:border-[#B9FF2C]/30 transition-all duration-300 hover:-translate-y-1"
            >
              <div className="w-10 h-10 rounded-xl bg-[#B9FF2C]/10 flex items-center justify-center mx-auto mb-4 group-hover:bg-[#B9FF2C]/20 transition-colors">
                <stat.icon className="w-5 h-5 text-[#B9FF2C]" />
              </div>
              <div className="text-[clamp(28px,4vw,36px)] font-black text-[#F4F6FA] mb-1">
                {stat.value}
              </div>
              <div className="text-[#A7ACB8] text-sm">
                {stat.label}
              </div>
            </div>
          ))}
        </div>

        {/* CTAs */}
        <div ref={ctaRef} className="flex flex-wrap items-center justify-center gap-4">
          <a
            href={`https://github.com/${REPO_OWNER}/${REPO_NAME}`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 bg-[#B9FF2C] text-[#07070A] px-6 py-3 rounded-full font-semibold hover:brightness-110 transition-all hover:-translate-y-0.5 hover:neon-box-glow"
          >
            <Github className="w-5 h-5" />
            View on GitHub
          </a>
          <a
            href={`https://github.com/${REPO_OWNER}/${REPO_NAME}/issues/new?template=build-request.yml`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 border border-white/20 text-[#F4F6FA] px-6 py-3 rounded-full font-medium hover:border-[#B9FF2C]/50 hover:text-[#B9FF2C] transition-all hover:-translate-y-0.5"
          >
            <MessageCircle className="w-5 h-5" />
            Request a Build
          </a>
        </div>
      </div>
    </section>
  );
}
