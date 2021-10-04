package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const (
	HOMEWORK_HELP = "**HOMEWORK HELP**:\n\n" +
		"  __**REQUEST HELP FOR A HOMEWORK:**__\n" +
		"```fix\n" +
		"crft homework “<PUT_WHAT_PROBLEM_YOU_NEED_HELP_HERE>”\n" +
		"```" +
		"This command will send you an ID for your request\n" +
		"__ex__: ***crft homework “need help for homework 2 question 2”***\n\n" +
		"__**GET A LIST OF USERS THAT CAN HELP YOU WITH YOUR PROBLEM:**__\n" +
		"```fix\n" +
		"crft curate <REQUEST_ID>" +
		"```" +
		"__ex__: ***crft curate 894069231174443028***\n\n" +
		"__**CONNECT WITH A USER FROM THE LIST TO GET HELP WITH YOUR PROBLEM:**__\n" +
		"```fix\n" +
		"crft connect <username> <REQUEST_ID>\n" +
		"```" +
		"__ex__:\n ***crft connect Thaight_ 894069231174443028***\n"
	SESSION_PLANNING = "**SESSION PLANNING:**\n\n" +
		"__**SCHEDULE A STUDY SESSION:**__\n" +
		"```diff\n" +
		"crft plan <DAY> <LOCATION> <TIME>\n" +
		"```" +
		"The location need to be conjoined\n" +
		"This command will send you an ID for your session planning\n" +
		"__ex__: ***crft plan Monday LoveLibrary 4:30pm***\n\n" +
		"__**GET A LIST OF POSSIBLE PARTICIPANTS AND NOTIFY THEM:**__\n" +
		"```diff\n" +
		"crft finalize <SESSION_ID>" +
		"```" +
		"__ex__: ***crft finalize 894069231174443028***"
)

var (
	Port  string
	Token string
)

func init() {
	Token = goDotEnvVariable("BOT_TOKEN")
	flag.StringVar(&Port, "p", "", "TCP Port")
	flag.Parse()

}

func main() {
	ln, err := net.Listen("tcp", Port)
	ln.Addr()
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	sess, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	sess.AddHandler(messageCreate)

	sess.Identify.Intents = discordgo.IntentsGuildMessages

	err = sess.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	sess.Close()
	conn.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	request := m.Content
	segment := strings.Split(request, " ")

	if segment[0] == "crft" {
		author := m.Author
		var mess *discordgo.Message

		if segment[1] == "homework" {
			var description string
			for _, msg := range segment[2:] {
				description += msg + " "
			}
			name := author.Username

			mess, _ = s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Description: name + " need help with homework\n" +
					"Description: " + description + "\n" +
					"\n\nReact with :one: if you're able to help.",
				Color: 0x000111,
			})

			s.MessageReactionAdd(m.ChannelID, mess.ID, "\x31\xef\xb8\x8f\xe2\x83\xa3")

			// Create a channel to send necessary information related to author request.
			channel, err := s.UserChannelCreate(m.Author.ID)
			if err != nil {
				fmt.Println("error creating channel:", err)
				s.ChannelMessageSend(
					m.ChannelID,
					"Something went wrong while sending the DM!",
				)
				return
			}
			_, err = s.ChannelMessageSendEmbed(channel.ID, &discordgo.MessageEmbed{
				Description: "You has requested for help with a homework\n" +
					"Here is your request Id: " + mess.ID,
				Color: 0xFFEE11,
			})
			if err != nil {
				fmt.Println("error sending DM message:", err)
				s.ChannelMessageSend(
					m.ChannelID,
					"Failed to send you a DM. "+
						"Did you disable DM in your privacy settings?",
				)
			}

		}

		if segment[1] == "curate" {
			messageId := segment[2]

			UsersList := curateUserList(s, m.ChannelID, messageId, "\x31\xef\xb8\x8f\xe2\x83\xa3")
			var helpers string
			for i, v := range UsersList {
				helpers += strconv.Itoa(i+1) + ". " + v.Username + "#" + v.Discriminator + "\n"
			}

			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Description: "List of users that can help you\n" + helpers,
			})

		}

		if segment[1] == "connect" {
			username := segment[2]
			messageId := segment[3]

			fmt.Println(messageId)
			UsersList := curateUserList(s, m.ChannelID, messageId, "\x31\xef\xb8\x8f\xe2\x83\xa3")

			for _, v := range UsersList {
				if v.Username == username {
					channel, err := s.UserChannelCreate(v.ID)
					if err != nil {
						fmt.Println("error creating channel:", err)
						s.ChannelMessageSend(
							m.ChannelID,
							"Something went wrong while sending the DM!",
						)
						return
					}
					_, err = s.ChannelMessageSendEmbed(channel.ID, &discordgo.MessageEmbed{
						Description: "**" + m.Author.String() + "**" + " has asked you for help with a homework problem!",
						Color:       0x444FEE,
					})
					if err != nil {
						fmt.Println("error sending DM message:", err)
						s.ChannelMessageSend(
							m.ChannelID,
							"Failed to send "+v.Username+" a DM. ",
						)
					}
				}
			}
		}

		if segment[1] == "plan" {
			when := segment[2]
			location := segment[3]
			time := segment[4]

			mess, _ := s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Description: "Date: " + when + "\nLocation: " + location + "\nTime: " + time +
					"\n\nReact with :one: if you're able to help.",
				Color: 0x00005441,
			})
			s.MessageReactionAdd(m.ChannelID, mess.ID, "\x31\xef\xb8\x8f\xe2\x83\xa3")

			channel, err := s.UserChannelCreate(m.Author.ID)
			if err != nil {
				fmt.Println("error creating channel:", err)
				s.ChannelMessageSend(
					m.ChannelID,
					"Something went wrong while sending the DM!",
				)
				return
			}
			_, err = s.ChannelMessageSendEmbed(channel.ID, &discordgo.MessageEmbed{
				Description: "You had scheduled a study session\n" +
					"Here is your session ID: " + mess.ID,
				Color: 0x00005441,
			})
			if err != nil {
				fmt.Println("error sending DM message:", err)
				s.ChannelMessageSend(
					m.ChannelID,
					"Failed to send you a DM. "+
						"Did you disable DM in your privacy settings?",
				)
			}
		}

		if segment[1] == "finalize" {
			sessionId := segment[2]

			ParticipantList := curateUserList(s, m.ChannelID, sessionId, "\x31\xef\xb8\x8f\xe2\x83\xa3")

			var curatedList string
			for i, v := range ParticipantList {
				curatedList += strconv.Itoa(i+1) + ". " + v.String() + "\n"
			}

			channel, err := s.UserChannelCreate(m.Author.ID)
			if err != nil {
				fmt.Println("error creating channel:", err)
				s.ChannelMessageSend(
					m.ChannelID,
					"Something went wrong while sending the DM!",
				)
				return
			}
			_, err = s.ChannelMessageSendEmbed(channel.ID, &discordgo.MessageEmbed{
				Description: "Here is the list of possible participant:\n" + curatedList,
				Color:       0x000FFE2,
			})
			if err != nil {
				fmt.Println("error sending DM message:", err)
				s.ChannelMessageSend(
					m.ChannelID,
					"Failed to send you a DM. "+
						"Did you disable DM in your privacy settings?",
				)
			}

			for _, v := range ParticipantList {
				if v.Bot {
					continue
				}
				channel, err := s.UserChannelCreate(v.ID)
				if err != nil {
					fmt.Println("error creating channel:", err)
					s.ChannelMessageSend(
						m.ChannelID,
						"Something went wrong while sending the DM!",
					)
					return
				}
				_, err = s.ChannelMessageSend(channel.ID, "You have an upcoming study session with ***"+m.Author.String()+"***")
				if err != nil {
					fmt.Println("error sending DM message:", err)
					s.ChannelMessageSend(
						m.ChannelID,
						"Failed to send you a DM. "+
							"Did you disable DM in your privacy settings?",
					)
				}

			}

		}

		if segment[1] == "help" {
			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Description: HOMEWORK_HELP,
				Color:       0x00FFE1E,
			})
			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Description: SESSION_PLANNING,
				Color:       0x00FFE1E,
			})
		}

		if segment[1] == "info" {
			author := m.Author.Username
			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Description: "Username: " + author + "\n",
				Color:       0x000FEEE,
				Image: &discordgo.MessageEmbedImage{
					URL: m.Author.AvatarURL("2048"),
				},
			})
		}
	}
}

//TODO: add a util function to curate a list of users react to a message.
func curateUserList(s *discordgo.Session, channelId, messageId, emoji string) []*discordgo.User {
	list, err := s.MessageReactions(channelId, messageId, emoji, 5, "", "")
	if err != nil {
		fmt.Println("error sending DM message:", err)
		s.ChannelMessageSendEmbed(channelId, &discordgo.MessageEmbed{
			Description: "Can not find your request message in the system\n",
			Color:       0x0001233,
		})
		return nil
	}
	return list
}

func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}
